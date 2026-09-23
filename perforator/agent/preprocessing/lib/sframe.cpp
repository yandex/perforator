#include "sframe.h"

#include <util/generic/maybe.h>
#include <perforator/lib/llvmex/llvm_exception.h>
#include <llvm/ADT/StringExtras.h>

#include <llvm/Object/SFrameParser.h>

namespace {
    struct TFunctionDescriptionEntry {
        int32_t Pc;
        uint32_t Size;
    };

    struct TFunctionRowEntry {
        uint8_t Info;
        uint32_t Pc;
        uint32_t Range;
        llvm::SmallVector<int32_t> Offsets;
    };

    struct TFdeFre {
        TFunctionDescriptionEntry Fde;
        TFunctionRowEntry Fre;
    };

    using TFreHandler = std::function<void(const TFdeFre&)>;

    template <llvm::endianness ESelectedEndian>
    void IterateFdeImpl(llvm::object::SectionRef sframeSection, TFreHandler handle) {
        using TParser = llvm::object::SFrameParser<ESelectedEndian>;

        auto view = Y_LLVM_RAISE(sframeSection.getContents());
        auto parser = Y_LLVM_RAISE(TParser::create(llvm::arrayRefFromStringRef(view), sframeSection.getAddress()));
        auto fdeRows = Y_LLVM_RAISE(parser.fdes());

        for (auto it = fdeRows.begin(); it != fdeRows.end(); ++it) {
            const auto& fde = *it;
            if (fde.Info.getFDEType() != llvm::sframe::FDEType::PCInc) {
                continue;
            }

            uint64_t fdeStartAddress = parser.getAbsoluteStartAddress(it);
            llvm::Error err = llvm::Error::success();

            // Collect pcs for ranges
            llvm::SmallVector<uint64_t, 4> pcs;
            for (const auto& fre : parser.fres(fde, err)) {
                pcs.push_back(fre.StartAddress);
            }
            pcs.push_back(fde.Size);
            if (err) {
                throw TLLVMException{} << toString(std::move(err));
            }

            // Run handler
            size_t i = 0;
            for (const auto& fre : parser.fres(fde, err)) {
                TFdeFre data;

                data.Fde.Pc = fdeStartAddress;
                data.Fde.Size = fde.Size;

                data.Fre.Info = fre.Info.Info.value();
                data.Fre.Pc = fdeStartAddress + fre.StartAddress;
                data.Fre.Range = pcs[i + 1] - pcs[i];
                data.Fre.Offsets = llvm::SmallVector<int32_t>(fre.Offsets.begin(), fre.Offsets.end());

                handle(data);
                i++;
            }
            if (err) {
                throw TLLVMException{} << toString(std::move(err));
            }
        }
    }

    template<llvm::endianness EEndian>
    bool CheckEndian(const char* sectionData) {
        using THeader = llvm::sframe::Header<EEndian>;
        THeader hdr;
        std::memcpy(&hdr, sectionData, sizeof(hdr));
        return hdr.Preamble.Magic.value() == llvm::sframe::Magic;
    }

    TMaybe<llvm::endianness> GetEndian(llvm::object::SectionRef sframeSection) {
        using THeader = llvm::sframe::Header<llvm::endianness::little>;
        Y_ENSURE(sframeSection.getSize() >= sizeof(THeader), "SFrame section smaller than SFrame Header.");

        auto view = Y_LLVM_RAISE(sframeSection.getContents());
        const char* sectionData = view.data();

        if (CheckEndian<llvm::endianness::little>(sectionData)) {
            return llvm::endianness::little;
        }
        if (CheckEndian<llvm::endianness::big>(sectionData)) {
            return llvm::endianness::big;
        }
        return {};
    }

    void HandleSframeSection(llvm::object::SectionRef sframeSection, TFreHandler handle) {
        auto maybeEndian = GetEndian(sframeSection);
        if (!maybeEndian) {
            return;
        }
        auto endian = *maybeEndian;
        if (endian == llvm::endianness::little) {
            IterateFdeImpl<llvm::endianness::little>(sframeSection, handle);
        } else {
            IterateFdeImpl<llvm::endianness::big>(sframeSection, handle);
        }
    }

    void IterateOverSframeFre(llvm::object::ObjectFile* objectFile, TFreHandler handle) {
        constexpr auto SFRAME_SECTION_NAME = llvm::StringRef(".sframe");
        for (auto section : objectFile->sections()) {
            auto maybeName = section.getName();
            if (!maybeName) {
                continue;
            }
            if (*maybeName == SFRAME_SECTION_NAME) {
                HandleSframeSection(section, handle);
                break;
            }
        }
    }

} // anonymous namespace

namespace NPerforator::NBinaryProcessing::NUnwind {

UnwindTable BuildUnwindTableFromSFrame(llvm::object::ObjectFile* objectFile, const NPerforator::NBinaryProcessing::BinaryAnalysisOptions& /*opts*/) {
    NUnwind::TRuleDictBuilder dictBuilder;
    NPerforator::NBinaryProcessing::NUnwind::UnwindTable unwtable;

    IterateOverSframeFre(objectFile, [&](const TFdeFre& fdefre) {
        const auto& fre = fdefre.Fre; // info, pc, range, vec<u32> offsets

        constexpr uint32_t SP_REG = 7; /* rsp */
        constexpr uint32_t FP_REG = 6; /* rbp */
        uint32_t baseReg = (fre.Info & 1) == 0 ? FP_REG : SP_REG;

        constexpr uint32_t CFA_OFFSET_IDX = 0;
        constexpr uint32_t FP_OFFSET_IDX = 1;

        /*
            https://www.sourceware.org/binutils/docs/sframe-spec.html#AMD64
            CFA = BASE_REG + offset1
            FP = CFA + offset2
            RA = -8
        */
        NUnwind::UnwindRule cfa;
        NUnwind::UnwindRule rbp;
        NUnwind::UnwindRule ra;

        unwtable.add_start_pc(fre.Pc);
        unwtable.add_pc_range(fre.Range);

        if (fre.Offsets.size() > CFA_OFFSET_IDX) {
            cfa.mutable_register_offset()->set_register_(baseReg);
            cfa.mutable_register_offset()->set_offset(fre.Offsets[CFA_OFFSET_IDX]);
        } else {
            cfa.mutable_unsupported();
        }
        if (fre.Offsets.size() > FP_OFFSET_IDX) {
            rbp.mutable_cfa_plus_offset()->set_offset(fre.Offsets[FP_OFFSET_IDX]);
            rbp.set_dereference(true);
        } else {
            rbp.mutable_unsupported();
        }
        ra.mutable_cfa_minus8();
        ra.set_dereference(true);

        unwtable.add_cfa(dictBuilder.Add(std::move(cfa)));
        unwtable.add_rbp(dictBuilder.Add(std::move(rbp)));
        unwtable.add_ra(dictBuilder.Add(std::move(ra)));
    });

    auto dict = std::move(dictBuilder).Finish();
    unwtable.mutable_dict()->Assign(dict.Rules().begin(), dict.Rules().end());
    RemapRules(unwtable.mutable_cfa(), dict);
    RemapRules(unwtable.mutable_rbp(), dict);
    RemapRules(unwtable.mutable_ra(), dict);

    // Sanity check.
    auto len = unwtable.start_pc_size();
    Y_ENSURE(len == unwtable.pc_range_size());
    Y_ENSURE(len == unwtable.cfa_size());
    Y_ENSURE(len == unwtable.rbp_size());
    Y_ENSURE(len == unwtable.ra_size());

    return unwtable;
}

} // namespace NPerforator::NBinaryProcessing::NUnwind
