#include "autofdo_input_builder.h"

#include <limits>
#include <string_view>
#include <stdexcept>

#include <fmt/format.h>

#include <llvm/Object/ELF.h>
#include <llvm/Object/ELFObjectFile.h>
#include <llvm/Object/ObjectFile.h>

#include <library/cpp/yt/compact_containers/compact_vector.h>

#include <perforator/lib/profile/profile.h>
#include <perforator/proto/profile/profile.pb.h>
#include <perforator/lib/llvmex/llvm_elf.h>

namespace NPerforator::NAutofdo {

namespace {

template <typename Map>
void MergeMap(Map& destination, const Map& source) {
    for (const auto& [k, v] : source) {
        destination[k] += v;
    }
}

struct TLoadSegment final {
    ui64 Vaddr{};
    ui64 FileOffset{};
    ui64 FileSize{};
};

class TBinaryAddressConverter final {
public:
    explicit TBinaryAddressConverter(TStringBuf binaryPath) {
        auto binary = llvm::object::ObjectFile::createObjectFile(binaryPath);
        if (!binary) {
            throw std::runtime_error{fmt::format(
                "Failed to read ELF binary '{}': {}",
                binaryPath,
                llvm::toString(binary.takeError())
            )};
        }

        const auto isElf = NLLVM::VisitELF(binary->getBinary(), [this] (const auto& elf) {
            BinarySize_ = elf.getELFFile().getBufSize();
            auto programHeaders = elf.getELFFile().program_headers();
            if (!programHeaders) {
                throw std::runtime_error{fmt::format(
                    "Failed to read ELF program headers: {}",
                    llvm::toString(programHeaders.takeError())
                )};
            }

            bool foundFirstLoad = false;
            for (const auto& phdr : *programHeaders) {
                if (phdr.p_type != llvm::ELF::PT_LOAD) {
                    continue;
                }
                if (!foundFirstLoad) {
                    FirstLoadVaddr_ = phdr.p_vaddr;
                    foundFirstLoad = true;
                }
                if ((phdr.p_flags & llvm::ELF::PF_X) != 0) {
                    ExecutableSegments_.push_back({phdr.p_vaddr, phdr.p_offset, phdr.p_filesz});
                }
            }

            if (!foundFirstLoad) {
                throw std::runtime_error{"ELF binary has no PT_LOAD segments"};
            }
            if (ExecutableSegments_.empty()) {
                throw std::runtime_error{"ELF binary has no executable PT_LOAD segments"};
            }

            return true;
        });
        if (!isElf) {
            throw std::runtime_error{"Profiled binary is not an ELF file"};
        }
    }

    ui64 GetFirstLoadVaddr() const {
        return FirstLoadVaddr_;
    }

    ui64 ToFileOffset(ui64 elfVaddr) const {
        for (const auto& segment : ExecutableSegments_) {
            if (elfVaddr < segment.Vaddr) {
                continue;
            }

            const ui64 delta = elfVaddr - segment.Vaddr;
            if (delta >= segment.FileSize ||
                delta > std::numeric_limits<ui64>::max() - segment.FileOffset) {
                continue;
            }

            const ui64 fileOffset = segment.FileOffset + delta;
            if (fileOffset >= BinarySize_) {
                throw std::runtime_error{fmt::format(
                    "ELF file offset {:#x} is outside binary of size {:#x}",
                    fileOffset,
                    BinarySize_
                )};
            }

            return fileOffset;
        }

        throw std::runtime_error{fmt::format(
            "ELF virtual address {:#x} is outside executable file data",
            elfVaddr
        )};
    }

private:
    ui64 BinarySize_{};
    ui64 FirstLoadVaddr_{};
    std::vector<TLoadSegment> ExecutableSegments_;
};

template <typename ELFT>
ui64 GetExecutableSectionsTotalSize(llvm::object::ObjectFile* file) {
    llvm::object::ELFObjectFile<ELFT>* elf = llvm::dyn_cast<llvm::object::ELFObjectFile<ELFT>>(file);
    if (!elf) {
        return 0;
    }

    ui64 totalSize{0};
    for (const auto& sectionRef : elf->sections()) {
        const auto elfSectionRef = static_cast<llvm::object::ELFSectionRef>(sectionRef);
        if (elfSectionRef.getType() == llvm::ELF::SHT_PROGBITS &&
            (elfSectionRef.getFlags() & llvm::ELF::SHF_EXECINSTR)) {
            totalSize += elfSectionRef.getSize();
        }
    }

    return totalSize;
}

using TPerforatorProfile = NPerforator::NProto::NProfile::Profile;
using TProfileReader = NPerforator::NProfile::TProfile;

[[noreturn]] void ThrowInvalidSampleError() {
    throw std::logic_error{"Invalid sample encountered, locations count is not even"};
}

}

ui64 GetBinaryInstructionsBytesSize(TStringBuf binaryPath) {
    auto binary = llvm::object::ObjectFile::createObjectFile(binaryPath);
    if (!binary) {
        return 0;
    }

    auto* file = binary->getBinary();
    #define TRY_ELF_TYPE(ELFT) \
    if (auto res = GetExecutableSectionsTotalSize<ELFT>(file)) { \
        return res; \
    }
    Y_LLVM_FOR_EACH_ELF_TYPE(TRY_ELF_TYPE)
    #undef TRY_ELF_TYPE

    return 0;
}

namespace {

template <typename ConvertAddress>
std::string SerializeAutofdoInputImpl(const TAutofdoInputData& data, ConvertAddress&& convertAddress) {
    std::string result{};
    result.reserve(16 * 1024 * 1024);

    fmt::format_to(std::back_inserter(result), "{}\n", data.RangeCountMap.size());
    for (const auto& [range, count] : data.RangeCountMap) {
        fmt::format_to(
            std::back_inserter(result),
            "{:#x}-{:#x}:{}\n",
            convertAddress(range.From),
            convertAddress(range.To),
            count
        );
    }

    fmt::format_to(std::back_inserter(result), "{}\n", data.AddressCountMap.size());
    for (const auto& [address, count] : data.AddressCountMap) {
        fmt::format_to(std::back_inserter(result), "{:#x}:{}\n", convertAddress(address), count);
    }

    fmt::format_to(std::back_inserter(result), "{}\n", data.BranchCountMap.size());
    for (const auto& [branch, count] : data.BranchCountMap) {
        fmt::format_to(
            std::back_inserter(result),
            "{:#x}->{:#x}:{}\n",
            convertAddress(branch.From),
            convertAddress(branch.To),
            count
        );
    }

    return result;
}

// The format description could be found here
// https://github.com/llvm/llvm-project/blob/release/18.x/bolt/include/bolt/Profile/DataAggregator.h#L389
//
// TODO : PERFORATOR-910, the format should be improved.
// See https://github.com/llvm/llvm-project/issues/149382#issuecomment-3085289377
std::string SerializeBoltInput(const TAutofdoInputData& data) {
    std::string result{};
    result.reserve(16 * 1024 * 1024);

    for (const auto& [branch, count] : data.BranchCountMap) {
        // Unfortunately we don't have "mispred_count" available, so set it to zero.
        fmt::format_to(std::back_inserter(result), "B {:x} {:x} {} 0\n",
            branch.From,
            branch.To,
            count
        );
    }
    for (const auto& [range, count] : data.RangeCountMap) {
        fmt::format_to(std::back_inserter(result), "F {:x} {:x} {}\n",
            range.From,
            range.To,
            count
        );
    }

    return result;
}

}

std::pair<std::string, std::string> SerializePGOInputsForBinary(
    const TAutofdoInputData& data,
    TStringBuf binaryPath
) {
    const TBinaryAddressConverter converter{binaryPath};

    return {
        SerializeAutofdoInputImpl(data, [&] (ui64 address) {
            return converter.ToFileOffset(address);
        }),
        SerializeBoltInput(data),
    };
}

///////////////////////////////////////////////////////////////////////////////////////////

TAutofdoInputData::TMetadata& TAutofdoInputData::TMetadata::operator+=(const TMetadata& other) {
    TotalProfiles += other.TotalProfiles;

    TotalBranches += other.TotalBranches;
    TotalSamples += other.TotalSamples;
    BogusLbrEntries += other.BogusLbrEntries;

    for (const auto& [service, count] : other.ProfilesCountByService) {
        ProfilesCountByService[service] += count;
    }

    return *this;
}

///////////////////////////////////////////////////////////////////////////////////////////

TInputBuilder::TInputBuilder(std::string buildId, ui64 skewedAddressAdjustment)
    : BuildId_{std::move(buildId)}
    , SkewedAddressAdjustment_{skewedAddressAdjustment}
{}

ui64 TInputBuilder::NormalizeAddress(ui64 address, bool isSkewed) const {
    if (!isSkewed) {
        return address;
    }
    if (address > std::numeric_limits<ui64>::max() - SkewedAddressAdjustment_) {
        throw std::runtime_error{fmt::format(
            "Profile address {:#x} overflows ELF virtual address",
            address
        )};
    }
    return address + SkewedAddressAdjustment_;
}

void TInputBuilder::AddProfile(std::string_view serviceName, TArrayRef<const char> profileBytes) {
    if (profileBytes.data() == nullptr || profileBytes.size() == 0) {
        return;
    }

    TPerforatorProfile profile{};
    if (!profile.ParseFromString(std::string_view{profileBytes.data(), profileBytes.size()})) {
        return;
    }

    AddProfile(serviceName, profile);
}

void TInputBuilder::AddProfile(std::string_view serviceName, const TPerforatorProfile& profile) {
    const TProfileReader profileReader{&profile};

    bool hasMainBinary = false;
    for (const auto binary : profileReader.Binaries()) {
        if (binary.GetBuildId().View() == BuildId_) {
            hasMainBinary = true;
            break;
        }
    }
    if (!hasMainBinary) {
        return;
    }
    auto& data = Data_;

    // The code below is an adaptation of how autofdo parses perf.data
    // https://github.com/google/autofdo/blob/3dafe34db0eb53af146cf782124f788ceaf6a9aa/sample_reader.cc#L292
    NYT::TCompactVector<TAutofdoInputData::TTakenBranch, 64> branchStack;
    for (const auto sample : profileReader.Samples()) {
        ++data.Meta.TotalSamples;

        branchStack.resize(0);
        TAutofdoInputData::TTakenBranch branch{};
        std::size_t frameIndex = 0;
        bool fromMainBinary = false;
        for (const auto stack : sample.GetKey().GetStacks()) {
            for (const auto frame : stack.GetFrames()) {
                const auto binary = frame.GetBinary();
                const bool mainBinary = binary.GetBuildId().View() == BuildId_;
                if (frameIndex++ % 2 == 0) {
                    fromMainBinary = mainBinary;
                    branch.From = mainBinary
                        ? NormalizeAddress(frame.GetAddress(), binary.HasSkewedAddresses())
                        : 0;
                } else {
                    if (fromMainBinary && mainBinary) {
                        branch.To = NormalizeAddress(frame.GetAddress(), binary.HasSkewedAddresses());
                    } else {
                        branch = {};
                    }
                    branchStack.push_back(branch);
                }
            }
        }
        if (frameIndex % 2 != 0) {
            ThrowInvalidSampleError();
        }
        if (branchStack.empty()) {
            continue;
        }

        if (branchStack[0].To != 0) {
            ++data.AddressCountMap[branchStack[0].To];
        }

        for (const auto& branch : branchStack) {
            if (branch.From != 0 && branch.To != 0) {
                ++data.BranchCountMap[branch];
                ++data.Meta.TotalBranches;
            }
        }

        for (std::size_t i = 1; i < branchStack.size(); ++i) {
            const auto begin = branchStack[i].To;
            const auto end = branchStack[i - 1].From;
            if (begin == 0 || end == 0) {
                continue;
            }
            // The interval between two taken branches shouldn't be too large
            if (end < begin || (end - begin > (1UL << 20))) {
                ++data.Meta.BogusLbrEntries;
                continue;
            }

            ++data.RangeCountMap[TAutofdoInputData::TRange{
                .From = begin,
                .To = end,
            }];
        }
    }

    ++data.Meta.TotalProfiles;
    ++data.Meta.ProfilesCountByService[TString{serviceName}];
}

void TInputBuilder::AddData(TAutofdoInputData&& otherData) {
    MergeMap(Data_.BranchCountMap, otherData.BranchCountMap);
    MergeMap(Data_.RangeCountMap, otherData.RangeCountMap);
    MergeMap(Data_.AddressCountMap, otherData.AddressCountMap);
    Data_.Meta += otherData.Meta;
}

TAutofdoInputData&& TInputBuilder::Finalize() && {
    return std::move(Data_);
}

///////////////////////////////////////////////////////////////////////////////////////////

TBatchInputBuilder::TBatchInputBuilder(ui64 buildersCount, std::string buildId, TStringBuf binaryPath) {
    const auto skewedAddressAdjustment = TBinaryAddressConverter{binaryPath}.GetFirstLoadVaddr();
    Builders_.reserve(buildersCount);
    for (std::size_t i = 0; i < buildersCount; ++i) {
        Builders_.emplace_back(buildId, skewedAddressAdjustment);
    }
}

TInputBuilder& TBatchInputBuilder::GetBuilder(ui64 builderIndex) {
    return Builders_.at(builderIndex);
}

TAutofdoInputData TBatchInputBuilder::Finalize() && {
    for (std::size_t i = 1; i < Builders_.size(); ++i) {
        Builders_[0].AddData(std::move(Builders_[i]).Finalize());
    }

    return std::move(Builders_[0]).Finalize();
}

///////////////////////////////////////////////////////////////////////////////////////////

void TBuildIdGuesser::FeedProfile(TArrayRef<const char> profileBytes) {
    if (profileBytes.data() == nullptr || profileBytes.size() == 0) {
        return;
    }

    TPerforatorProfile profile{};
    if (!profile.ParseFromString(std::string_view{profileBytes.data(), profileBytes.size()})) {
        return;
    }

    FeedProfile(profile);
}

void TBuildIdGuesser::FeedProfile(const TPerforatorProfile& profile) {
    const TProfileReader profileReader{&profile};
    for (const auto sample : profileReader.Samples()) {
        for (const auto stack : sample.GetKey().GetStacks()) {
            for (const auto frame : stack.GetFrames()) {
                ++BuildIdCount_[std::string{frame.GetBinary().GetBuildId().View()}];
            }
        }
    }
}

const absl::flat_hash_map<std::string, ui64>& TBuildIdGuesser::GetFrequencyMap() const {
    return BuildIdCount_;
}

TBatchBuildIdGuesser::TBatchBuildIdGuesser(ui64 guessersCount) {
    Guessers_.resize(guessersCount);
}

TBuildIdGuesser& TBatchBuildIdGuesser::GetGuesser(ui64 guesserIndex) {
    return Guessers_.at(guesserIndex);
}

std::optional<std::string> TBatchBuildIdGuesser::GuessBuildID() const {
    absl::flat_hash_map<std::string, ui64> totalBuildIdCount;
    for (const auto& guesser : Guessers_) {
        MergeMap(totalBuildIdCount, guesser.GetFrequencyMap());
    }

    std::optional<std::string> mostFrequentId{};
    ui64 currentBest = 0;

    for (const auto& [k, v] : totalBuildIdCount) {
        if (v > currentBest) {
            mostFrequentId.emplace(k);
            currentBest = v;
        }
    }

    return mostFrequentId;
}

}
