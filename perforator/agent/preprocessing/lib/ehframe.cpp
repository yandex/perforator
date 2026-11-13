#include "ehframe.h"
#include "dwarf_matcher.h"

#include <perforator/lib/llvmex/llvm_exception.h>
#include <library/cpp/iterator/enumerate.h>
#include <util/generic/maybe.h>

#include <llvm/ADT/AddressRanges.h>
#include <llvm/DebugInfo/DWARF/DWARFContext.h>
#include <llvm/DebugInfo/DWARF/DWARFDebugFrame.h>

namespace NPerforator::NBinaryProcessing::NUnwind {

////////////////////////////////////////////////////////////////////////////////

// Ugly hack to make pointer to the private member.
// UnwindLocation::Dereference is private and there is no getter for it.
// See http://bloglitb.blogspot.com/2011/12/access-to-private-members-safer.html?m=1
template <typename Tag, typename Tag::type M>
struct Backdoor {
    friend typename Tag::type get(Tag) {
        return M;
    }
};

template <typename Tag, typename Member>
struct TagBase {
    typedef Member type;
    friend type get(Tag);
};

struct UnwindLocationTag : TagBase<UnwindLocationTag, bool llvm::dwarf::UnwindLocation::*> {};

template struct Backdoor<UnwindLocationTag, &llvm::dwarf::UnwindLocation::Dereference>;

bool IsDerefLocation(const llvm::dwarf::UnwindLocation& loc) {
    return loc.*get(UnwindLocationTag{});
}

////////////////////////////////////////////////////////////////////////////////

// Mapping from DWARF registers numbers to actual registers, according to ABI
#ifdef __x86_64__
// See: https://refspecs.linuxbase.org/elf/x86_64-abi-0.99.pdf, Figure 3.36
static constexpr uint32_t kFrameRegister = 6; // rbp
#elif __aarch64__
// See: https://github.com/ARM-software/abi-aa/blob/c51addc3dc03e73a016a1e4edf25440bcac76431/aadwarf64/aadwarf64.rst#41dwarf-register-names
static constexpr uint32_t kFrameRegister = 29; // frame register
#else
#error This arch is not supported by Perforator yet
#endif

////////////////////////////////////////////////////////////////////////////////

// Matches CFA = *(reg + offset) + bias, where reg is the stack or the frame register.
static bool MatchRegisterDerefOffset(const llvm::DWARFExpression& expr, UnwindRule* rule) {
    for (int regno : {6, 7}) {
        auto atom = static_cast<llvm::dwarf::LocationAtom>(llvm::dwarf::DW_OP_breg0 + regno);

        ui64 offset = 0;
        ui64 bias = 0;
        bool matches = TDwarfExpressionPattern{}
            .Push(atom, Wildcard(&offset))
            .Push(llvm::dwarf::DW_OP_deref)
            .Push(llvm::dwarf::DW_OP_plus_uconst, Wildcard(&bias))
            .Matches(expr);

        if (!matches) {
            bias = 0;
            matches = TDwarfExpressionPattern{}
                .Push(atom, Wildcard(&offset))
                .Push(llvm::dwarf::DW_OP_deref)
                .Matches(expr);
        }

        if (matches) {
            auto* deref = rule->mutable_register_deref_offset();
            deref->set_register_(regno);
            deref->set_offset(static_cast<i64>(offset));
            deref->set_bias(bias);
            return true;
        }
    }
    return false;
}

// Matches reg + offset, where reg is the stack or the frame register.
static bool MatchRegisterOffset(const llvm::DWARFExpression& expr, UnwindRule* rule) {
    for (int regno : {6, 7}) {
        auto atom = static_cast<llvm::dwarf::LocationAtom>(llvm::dwarf::DW_OP_breg0 + regno);

        ui64 offset = 0;
        if (TDwarfExpressionPattern{}.Push(atom, Wildcard(&offset)).Matches(expr)) {
            rule->mutable_register_offset()->set_register_(regno);
            rule->mutable_register_offset()->set_offset(static_cast<i64>(offset));
            return true;
        }
    }
    return false;
}

// Matches CFA = *(rsp + offset + index * 8) + bias for any index register.
static bool MatchRegisterIndexedDerefOffset(const llvm::DWARFExpression& expr, UnwindRule* rule) {
    for (int index = 0; index < 16; ++index) {
        auto indexAtom = static_cast<llvm::dwarf::LocationAtom>(llvm::dwarf::DW_OP_breg0 + index);

        ui64 offset = 0;
        ui64 bias = 0;
        bool matches = TDwarfExpressionPattern{}
            .Push(llvm::dwarf::DW_OP_breg7, Wildcard(&offset))
            .Push(indexAtom, 0)
            .Push(llvm::dwarf::DW_OP_lit8)
            .Push(llvm::dwarf::DW_OP_mul)
            .Push(llvm::dwarf::DW_OP_plus)
            .Push(llvm::dwarf::DW_OP_deref)
            .Push(llvm::dwarf::DW_OP_plus_uconst, Wildcard(&bias))
            .Matches(expr);

        if (matches) {
            auto* deref = rule->mutable_register_indexed_deref_offset();
            deref->set_register_(7);
            deref->set_offset(static_cast<i64>(offset));
            deref->set_index_register(index);
            deref->set_scale(8);
            deref->set_bias(bias);
            return true;
        }
    }
    return false;
}

void FillRegisterLocation(UnwindRule* rule, const llvm::dwarf::UnwindLocation& loc) {
    rule->set_dereference(IsDerefLocation(loc));

    switch (loc.getLocation()) {
    case llvm::dwarf::UnwindLocation::Unspecified:
    case llvm::dwarf::UnwindLocation::Undefined:
    case llvm::dwarf::UnwindLocation::Same:
        // These are meaningless for unwinding
        rule->mutable_unsupported();
        break;
    case llvm::dwarf::UnwindLocation::CFAPlusOffset:
        if (loc.getOffset() == -8) {
            rule->mutable_cfa_minus8();
        } else {
            rule->mutable_cfa_plus_offset()->set_offset(loc.getOffset());
        }
        break;
    case llvm::dwarf::UnwindLocation::RegPlusOffset:
        rule->mutable_register_offset()->set_register_(loc.getRegister());
        rule->mutable_register_offset()->set_offset(loc.getOffset());
        break;
    case llvm::dwarf::UnwindLocation::DWARFExpr: {
        auto expr = loc.getDWARFExpressionBytes();
        // Tricky part: support some common dwarf expressions.

        {
            // 1. Static hand-written CFA rule for .plt sections:
            // DW_OP_breg7 RSP+8, DW_OP_breg16 RIP+0, DW_OP_lit15, DW_OP_and, DW_OP_lit10, DW_OP_ge, DW_OP_lit3, DW_OP_shl, DW_OP_plus

            bool matches = NPerforator::NBinaryProcessing::NUnwind::TDwarfExpressionPattern{}
                .Push(llvm::dwarf::DW_OP_breg7, 8)
                .Push(llvm::dwarf::DW_OP_breg16, 0)
                .Push(llvm::dwarf::DW_OP_lit15)
                .Push(llvm::dwarf::DW_OP_and)
                .Push(llvm::dwarf::DW_OP_lit10)
                .Push(llvm::dwarf::DW_OP_ge)
                .Push(llvm::dwarf::DW_OP_lit3)
                .Push(llvm::dwarf::DW_OP_shl)
                .Push(llvm::dwarf::DW_OP_plus)
                .Matches(*expr);

            if (matches) {
                rule->mutable_plt_section();
                break;
            }
        }

        // 2. CFA saved on the stack:
        // DW_OP_breg[67] R[BS]P+544, DW_OP_deref, DW_OP_plus_uconst 0x8 (hand-written, openssl)
        // DW_OP_breg6 RBP-8, DW_OP_deref (gcc stack realignment through DRAP)
        if (MatchRegisterDerefOffset(*expr, rule)) {
            break;
        }

        // 3. Register saved at an address held in a register (DW_CFA_expression, so dereference is set):
        // DW_OP_breg6 RBP+0 for rbp (gcc stack realignment through DRAP: push %rbp; mov %rsp, %rbp)
        if (MatchRegisterOffset(*expr, rule)) {
            break;
        }

        // 4. CFA saved above a variable-size stack area (hand-written, openssl bn_mul_mont and friends):
        // DW_OP_breg7 RSP+8, DW_OP_breg9 R9+0, DW_OP_lit8, DW_OP_mul, DW_OP_plus, DW_OP_deref, DW_OP_plus_uconst 0x8
        MatchRegisterIndexedDerefOffset(*expr, rule);
        break;
    }
    case llvm::dwarf::UnwindLocation::Constant:
        rule->mutable_constant()->set_value(loc.getConstant());
        break;
    // no explicit default to trigger compilation error on new location kind
    }
}

////////////////////////////////////////////////////////////////////////////////

template<typename T>
class TAddressRangeMapBuilder {
public:
    TAddressRangeMapBuilder() = default;

    TAddressRangeMapBuilder(size_t capacity) {
        Values_.reserve(capacity);
        Events_.reserve(capacity * 2);
    }

    void Insert(llvm::AddressRange range, T value) {
        size_t index = Values_.size();
        Values_.push_back(std::move(value));
        Events_.push_back({range.start(), index});
        Events_.push_back({range.end(), index});
    }

    std::vector<std::pair<llvm::AddressRange, T>> Finish() && {
        std::vector<std::pair<llvm::AddressRange, T>> res;

        Sort(Events_);

        std::set<size_t> activeIndices;
        uint64_t startAddress = 0;
        std::optional<size_t> currentIndex;

        auto updateCurrent = [&](uint64_t address) {
            std::optional<size_t> newIndex;
            if (!activeIndices.empty()) {
                newIndex = *activeIndices.begin();
            }
            if (currentIndex != newIndex) {
                if (currentIndex.has_value()) {
                    res.emplace_back(llvm::AddressRange{startAddress, address}, Values_[*currentIndex]);
                }
                currentIndex = newIndex;
                startAddress = address;
            }
        };

        for (auto&& [address, index] : Events_) {
            auto it = activeIndices.lower_bound(index);
            if (it != activeIndices.end() && *it == index) {
                activeIndices.erase(it);
                updateCurrent(address);
            } else {
                activeIndices.insert(it, index);
                updateCurrent(address);
            }
        }

        return res;
    }

private:
    std::vector<T> Values_;
    std::vector<std::pair<uint64_t, size_t>> Events_;
};

UnwindTable BuildUnwindTableFromEhFrame(llvm::object::ObjectFile* objectFile, const NPerforator::NBinaryProcessing::BinaryAnalysisOptions& /*opts*/) {
    auto dwarfContext = llvm::DWARFContext::create(*objectFile);

    bool isEh = true;
    const llvm::DWARFDebugFrame* ehFrame = nullptr;
    if (isEh) {
        ehFrame = Y_LLVM_RAISE(dwarfContext->getEHFrame());
    } else {
        ehFrame = Y_LLVM_RAISE(dwarfContext->getDebugFrame());
    }
    Y_ENSURE(ehFrame);

    // Some DWARFs (of BOLTed binaries, in particular), can contain overlapping ranges.
    // We try to mimic libunwind behavior and choose first matching FDE for pc.
    // It is better to just use some kind of interval_map, but llvm::AddressRangesMap is backed by SmallVector and can cause performance issues.
    auto entries = ehFrame->entries();
    TAddressRangeMapBuilder<const llvm::dwarf::FDE*> fdeRangesMap;
    for (auto&& entry : entries) {
        const llvm::dwarf::FDE* fde = llvm::dyn_cast<llvm::dwarf::FDE>(&entry);
        if (fde == nullptr) {
            Y_ENSURE(llvm::isa<llvm::dwarf::CIE>(&entry), "Unknown eh frame kind " << (int)entry.getKind());
            continue;
        }
        fdeRangesMap.Insert(llvm::AddressRange{fde->getInitialLocation(), fde->getInitialLocation() + fde->getAddressRange()}, fde);
    }
    auto fdeEntries = std::move(fdeRangesMap).Finish();

    NUnwind::TRuleDictBuilder dictBuilder;
    NPerforator::NBinaryProcessing::NUnwind::UnwindTable unwtable;

    for (auto&& [fdeAddressRange, fde] : fdeEntries) {
        const llvm::dwarf::CIE* cie = fde->getLinkedCIE();
        Y_ENSURE(cie, "Empty CIE for FDE at " << fde->getOffset());

        llvm::dwarf::UnwindTable table = Y_LLVM_RAISE(llvm::dwarf::createUnwindTable(fde));

        for (const auto& [i, row] : Enumerate(table)) {
            auto rowAddressRange = [&]() -> TMaybe<llvm::AddressRange> {
                auto startAddress = row.getAddress();
                auto endAddress = [&]() -> uint64_t {
                    if (i + 1 < table.size()) {
                        return table[i + 1].getAddress();
                    }
                        return fde->getInitialLocation() + fde->getAddressRange();
                }();

                Y_ENSURE(endAddress >= startAddress);
                // We met DWARFs with fde->getInitialLocation() + fde->getAddressRange() == row.Address(),
                // however according to section 6.4.1 of DWARF spec it is incorrect.
                // Nonetheless, we still give best efforts to process these binaries.
                if (startAddress == endAddress) {
                    return Nothing();
                }

                // Bind range to precalculated boundaries of FDE (if FDEs do overlap).
                startAddress = Max(startAddress, fdeAddressRange.start());
                endAddress = Min(endAddress, fdeAddressRange.end());
                if (startAddress >= endAddress) {
                    return Nothing();
                }

                return llvm::AddressRange{startAddress, endAddress};
            }();

            if (!rowAddressRange) {
                continue;
            }
            unwtable.add_start_pc(rowAddressRange->start());
            unwtable.add_pc_range(rowAddressRange->size());

            NUnwind::UnwindRule cfa;
            NUnwind::UnwindRule rbp;
            NUnwind::UnwindRule ra;

            auto loc = row.getRegisterLocations().getRegisterLocation(cie->getReturnAddressRegister());
            if (loc) {
                ra.set_dereference(IsDerefLocation(*loc));

                switch (loc.value().getLocation()) {
                case llvm::dwarf::UnwindLocation::Undefined:
                    ra.mutable_undefined();
                    break;
                case llvm::dwarf::UnwindLocation::Unspecified:
                case llvm::dwarf::UnwindLocation::Same:
                    // These are meaningless for unwinding
                    ra.mutable_unsupported();
                    break;
                case llvm::dwarf::UnwindLocation::CFAPlusOffset:
                    if (loc->getOffset() == -8) {
                        ra.mutable_cfa_minus8();
                    } else {
                        ra.mutable_cfa_plus_offset()->set_offset(loc->getOffset());
                    }
                    break;
                case llvm::dwarf::UnwindLocation::RegPlusOffset:
                    ra.mutable_register_offset()->set_register_(loc->getRegister());
                    ra.mutable_register_offset()->set_offset(loc->getOffset());
                    break;
                case llvm::dwarf::UnwindLocation::DWARFExpr:
                    // We do not support dwarf expressions here.
                    ra.mutable_unsupported();
                    break;
                case llvm::dwarf::UnwindLocation::Constant:
                    ra.mutable_constant()->set_value(loc->getConstant());
                    break;
                // no explicit default to trigger compilation error on new location kind
                }
            } else {
                // If we found no location for aarch64/riscv
                // Then it means that current RA is valid for current stack frame
                // And there is no point of adressing it from FDE
                ra.mutable_unsupported();
            }

            // Fill CFA value.
            FillRegisterLocation(&cfa, row.getCFAValue());

            // Fill RBP/r29 value. It is needed in ~25% of unwind rules.
            if (auto loc = row.getRegisterLocations().getRegisterLocation(kFrameRegister)) {
                FillRegisterLocation(&rbp, *loc);
            } else {
                rbp.mutable_unsupported();
            }

            unwtable.add_cfa(dictBuilder.Add(std::move(cfa)));
            unwtable.add_rbp(dictBuilder.Add(std::move(rbp)));
            unwtable.add_ra(dictBuilder.Add(std::move(ra)));
        }
    }

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

////////////////////////////////////////////////////////////////////////////////

} // namespace NPerforator::NBinaryProcessing::NUnwind
