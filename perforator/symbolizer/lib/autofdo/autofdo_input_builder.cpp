#include "autofdo_input_builder.h"

#include <string_view>
#include <stdexcept>

#include <fmt/format.h>

#include <llvm/Object/ELF.h>
#include <llvm/Object/ELFObjectFile.h>
#include <llvm/Object/ObjectFile.h>

#include <library/cpp/yt/compact_containers/compact_vector.h>

#include <perforator/proto/pprofprofile/profile.pb.h>
#include <perforator/lib/llvmex/llvm_elf.h>


namespace NPerforator::NAutofdo {

namespace {

template <typename Map>
void MergeMap(Map& destination, const Map& source) {
    for (const auto& [k, v] : source) {
        destination[k] += v;
    }
}

const std::string kEmptyString{};

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

template <typename Traits>
class TItemByIdMap final {
public:
    using value_type = typename Traits::value_type;

    explicit TItemByIdMap(const NPerforator::NProto::NPProf::Profile& profile) {
        SmallIdMap_.assign(Traits::Size(profile) + 1, nullptr);

        for (std::size_t i = 0; i < Traits::Size(profile); ++i) {
            const auto& item = Traits::At(profile, i);
            const auto itemId = item.id();
            if (itemId < SmallIdMap_.size()) {
                SmallIdMap_[itemId] = &item;
            } else {
                BigIdMap_.emplace(itemId, &item);
            }
        }
    }

    const value_type& At(ui64 itemId) const {
        const value_type* itemPtr = nullptr;

        if (itemId < SmallIdMap_.size()) {
            itemPtr = SmallIdMap_[itemId];
        } else {
            itemPtr = BigIdMap_.at(itemId);
        }

        if (itemPtr == nullptr) {
            throw std::logic_error{fmt::format("No item with id {}", itemId)};
        }

        return *itemPtr;
    }

private:
    std::vector<const value_type*> SmallIdMap_;
    absl::flat_hash_map<ui64, const value_type*> BigIdMap_;
};

struct LocationByIdTraits final {
    using value_type = NPerforator::NProto::NPProf::Location;

    static std::size_t Size(const NPerforator::NProto::NPProf::Profile& profile) {
        return profile.locationSize();
    }

    static const value_type& At(const NPerforator::NProto::NPProf::Profile& profile, std::size_t i) {
        return profile.location(i);
    }
};

struct MappingByIdTraits final {
    using value_type = NPerforator::NProto::NPProf::Mapping;

    static std::size_t Size(const NPerforator::NProto::NPProf::Profile& profile) {
        return profile.mappingSize();
    }

    static const value_type& At(const NPerforator::NProto::NPProf::Profile& profile, std::size_t i) {
        return profile.mapping(i);
    }
};

using TLocationByIdMap = TItemByIdMap<LocationByIdTraits>;
using TMappingByIdMap = TItemByIdMap<MappingByIdTraits>;

const std::string& GetBuildId(
    const TMappingByIdMap& mappingById,
    const NPerforator::NProto::NPProf::Profile& profile,
    ui64 mappingId) {
    if (mappingId == 0) {
        return kEmptyString;
    }

    const auto& mapping = mappingById.At(mappingId);
    return profile.string_table(mapping.build_id());
};

TMappingByIdMap PrepareProfileMappings(const NPerforator::NProto::NPProf::Profile& profile) {
    return TMappingByIdMap{profile};
}

TLocationByIdMap PrepareProfileLocations(const NPerforator::NProto::NPProf::Profile& profile) {
    return TLocationByIdMap{profile};
}

std::optional<ui64> PrepareMainMappingId(const NPerforator::NProto::NPProf::Profile& profile, const std::string& buildId) {
    for (std::size_t i = 0; i < profile.mappingSize(); ++i) {
        const auto& mapping = profile.mapping(i);
        if (profile.string_table(mapping.build_id()) == buildId) {
            return mapping.id();
        }
    }

    return std::nullopt;
}

ui64 PrepareMainMappingOffset(const NPerforator::NProto::NPProf::Profile& profile, const std::string& buildId) {
    for (std::size_t i = 0; i < profile.mappingSize(); ++i) {
        const auto& mapping = profile.mapping(i);
        if (profile.string_table(mapping.build_id()) == buildId) {
            return mapping.memory_start() - mapping.file_offset();
        }
    }

    return 0;
}

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

std::string SerializeAutofdoInput(const TAutofdoInputData& data) {
    std::string result{};
    result.reserve(16 * 1024 * 1024);

    fmt::format_to(std::back_inserter(result), "{}\n", data.RangeCountMap.size());
    for (const auto& [range, count] : data.RangeCountMap) {
        fmt::format_to(std::back_inserter(result), "{:#x}-{:#x}:{}\n", range.From, range.To, count);
    }

    fmt::format_to(std::back_inserter(result), "{}\n", data.AddressCountMap.size());
    for (const auto& [address, count] : data.AddressCountMap) {
        fmt::format_to(std::back_inserter(result), "{:#x}:{}\n", address, count);
    }

    fmt::format_to(std::back_inserter(result), "{}\n", data.BranchCountMap.size());
    for (const auto& [branch, count] : data.BranchCountMap) {
        fmt::format_to(std::back_inserter(result), "{:#x}->{:#x}:{}\n", branch.From, branch.To, count);
    }

    return result;
}

// The format description could be found here
// https://github.com/llvm/llvm-project/blob/release/18.x/bolt/include/bolt/Profile/DataAggregator.h#L389
std::string SerializeAutofdoInputInBoltPreaggregatedFormat(const TAutofdoInputData& data) {
    std::string result{};
    result.reserve(16 * 1024 * 1024);

    for (const auto& [branch, count] : data.BranchCountMap) {
        // Unfortunately we don't have "mispred_count" available, so set it to zero.
        fmt::format_to(std::back_inserter(result), "B {:x} {:x} {} 0\n",
            branch.From + branch.MappingOffset,
            branch.To + branch.MappingOffset,
            count
        );
    }
    for (const auto& [range, count] : data.RangeCountMap) {
        fmt::format_to(std::back_inserter(result), "F {:x} {:x} {}\n",
            range.From + range.MappingOffset,
            range.To + range.MappingOffset,
            count
        );
    }

    return result;
}

///////////////////////////////////////////////////////////////////////////////////////////

TAutofdoInputData::TMetadata& TAutofdoInputData::TMetadata::operator+=(const TMetadata& other) {
    TotalProfiles += other.TotalProfiles;

    TotalBranches += other.TotalBranches;
    TotalSamples += other.TotalSamples;
    BogusLbrEntries += other.BogusLbrEntries;

    return *this;
}

///////////////////////////////////////////////////////////////////////////////////////////

TInputBuilder::TInputBuilder(const std::string& buildId) : BuildId_{buildId} {}

void TInputBuilder::AddProfile(TArrayRef<const char> profileBytes) {
    if (profileBytes.data() == nullptr || profileBytes.size() == 0) {
        return;
    }

    NPerforator::NProto::NPProf::Profile profile{};
    if (!profile.ParseFromString(std::string_view{profileBytes.data(), profileBytes.size()})) {
        return;
    }

    AddProfile(profile);
}

void TInputBuilder::AddProfile(const NPerforator::NProto::NPProf::Profile& profile) {
    const auto locationById = PrepareProfileLocations(profile);
    const auto mappingById = PrepareProfileMappings(profile);
    const auto mainMappingIdOpt = PrepareMainMappingId(profile, BuildId_);
    if (!mainMappingIdOpt.has_value()) {
        return;
    }
    const auto mainMappingId = *mainMappingIdOpt;
    const auto mainMappingOffset = PrepareMainMappingOffset(profile, BuildId_);

    const auto calcLocationAddress = [&mappingById] (const NPerforator::NProto::NPProf::Location& loc) -> ui64 {
        if (loc.mapping_id() == 0) {
            return 0;
        }

        const auto& mapping = mappingById.At(loc.mapping_id());
        return static_cast<ui64>(mapping.file_offset()) + (loc.address() - mapping.memory_start());
    };

    // The code below is an adaptation of how autofdo parses perf.data
    // https://github.com/google/autofdo/blob/3dafe34db0eb53af146cf782124f788ceaf6a9aa/sample_reader.cc#L292
    NYT::TCompactVector<TAutofdoInputData::TTakenBranch, 64> branchStack;
    for (std::size_t i = 0; i < profile.sampleSize(); ++i) {
        const auto& sample = profile.sample(i);
        if (sample.location_idSize() % 2 != 0) {
            ThrowInvalidSampleError();
        }

        ++Data_.Meta.TotalSamples;

        branchStack.resize(0);
        for (std::size_t j = 0; j < sample.location_idSize(); j += 2) {
            const auto& locFrom = locationById.At(sample.location_id(j));
            const auto& locTo = locationById.At(sample.location_id(j + 1));

            if (locFrom.mapping_id() == mainMappingId && locTo.mapping_id() == mainMappingId) {
                branchStack.push_back(TAutofdoInputData::TTakenBranch{
                    .From = calcLocationAddress(locFrom),
                    .To = calcLocationAddress(locTo),
                    .MappingOffset = mainMappingOffset,
                });
            } else {
                branchStack.push_back(TAutofdoInputData::TTakenBranch{0, 0, 0});
            }
        }
        if (branchStack.empty()) {
            continue;
        }

        if (branchStack[0].To != 0) {
            ++Data_.AddressCountMap[branchStack[0].To];
        }

        for (const auto& branch : branchStack) {
            if (branch.From != 0 && branch.To != 0) {
                ++Data_.BranchCountMap[branch];
                ++Data_.Meta.TotalBranches;
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
                ++Data_.Meta.BogusLbrEntries;
                continue;
            }

            ++Data_.RangeCountMap[TAutofdoInputData::TRange{
                .From = begin,
                .To = end,
                .MappingOffset = branchStack[i].MappingOffset,
            }];
        }
    }

    ++Data_.Meta.TotalProfiles;
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

TBatchInputBuilder::TBatchInputBuilder(ui64 buildersCount, std::string buildId) {
    Builders_.reserve(buildersCount);
    for (std::size_t i = 0; i < buildersCount; ++i) {
        Builders_.emplace_back(buildId);
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

namespace {

class TMappingsCounter final {
public:
    explicit TMappingsCounter(const NPerforator::NProto::NPProf::Profile& profile) : Profile_{profile} {
        SmallIdCounter_.assign(Profile_.mappingSize() + 1, 0);
    }

    void Increment(ui64 mappingId) {
        if (mappingId < SmallIdCounter_.size()) {
            ++SmallIdCounter_[mappingId];
        } else {
            ++BigIdCounter_[mappingId];
        }
    }

    void SinkBuildIdFrequenciesInto(absl::flat_hash_map<std::string, ui64>& dst) && {
        const auto mappingById = PrepareProfileMappings(Profile_);
        const auto getBuildId = [&mappingById, this] (ui64 mappingId) -> const std::string& {
            return GetBuildId(mappingById, Profile_, mappingId);
        };

        for (std::size_t mappingId = 0; mappingId < SmallIdCounter_.size(); ++mappingId) {
            const auto count = SmallIdCounter_[mappingId];
            if (count == 0) {
                continue;
            }

            dst[getBuildId(mappingId)] += count;
        }

        for (const auto& [mappingId, count] : BigIdCounter_) {
            dst[getBuildId(mappingId)] += count;
        }
    }

private:
    const NPerforator::NProto::NPProf::Profile& Profile_;

    std::vector<ui64> SmallIdCounter_;
    absl::flat_hash_map<ui64, ui64> BigIdCounter_;
};

}

void TBuildIdGuesser::FeedProfile(TArrayRef<const char> profileBytes) {
    if (profileBytes.data() == nullptr || profileBytes.size() == 0) {
        return;
    }

    NPerforator::NProto::NPProf::Profile profile{};
    if (!profile.ParseFromString(std::string_view{profileBytes.data(), profileBytes.size()})) {
        return;
    }

    FeedProfile(profile);
}

void TBuildIdGuesser::FeedProfile(const NPerforator::NProto::NPProf::Profile& profile) {
    TMappingsCounter mappingsCounter{profile};

    const auto locationById = PrepareProfileLocations(profile);

    for (std::size_t i = 0; i < profile.sampleSize(); ++i) {
        const auto& sample = profile.sample(i);
        if (sample.location_idSize() % 2 != 0) {
            ThrowInvalidSampleError();
        }

        for (std::size_t j = 0; j < sample.location_idSize(); ++j) {
            const auto& loc = locationById.At(sample.location_id(j));

            mappingsCounter.Increment(loc.mapping_id());
        }
    }

    std::move(mappingsCounter).SinkBuildIdFrequenciesInto(BuildIdCount_);
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
