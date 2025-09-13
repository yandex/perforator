#include "service_perf_top_aggregator.h"

#include <perforator/symbolizer/lib/symbolize/symbolizer.h>

#include <library/cpp/yt/compact_containers/compact_vector.h>

#include <algorithm>
#include <string_view>

namespace NPerforator::NClusterTop {

namespace {

template<typename T>
using TSmallVector = NYT::TCompactVector<T, 4>;

constexpr std::size_t kMaxEntriesToPrint = 10'000;

constexpr std::string_view kCPUCyclesType = "cpu";
constexpr std::string_view kCPUCyclesUnit = "cycles";

constexpr std::string_view kUnknownFunction = "<UNKNOWN (No function)>";
constexpr std::string_view kUnknownMapping = "<UNKNOWN (No mapping)>";
constexpr std::string_view kUnknownBuildId = "<UNKNOWN (No buildId)>";
constexpr std::string_view kUnknownNoFrames = "<UNKNOWN (No frames)>";
constexpr std::string_view kNoGSYMLocation = "<UNKNOWN (No GSYM)>";

constexpr std::string_view kKernelSpecialMapping = "[kernel]";

constexpr std::size_t kMaxSymbolizationCacheTotalSize = 256 * 1024;

ui64 GetCpuCyclesValue(
    const NPerforator::NProto::NPProf::ProfileLight& profile,
    const NPerforator::NProto::NPProf::SampleLight& sample) {
    for (std::size_t i = 0; i < profile.sample_typeSize(); ++i) {
        const auto& sampleType = profile.sample_type(i);
        if (profile.string_table(sampleType.type()) == kCPUCyclesType &&
            profile.string_table(sampleType.unit()) == kCPUCyclesUnit) {
            return sample.value(i);
        }
    }

    return 0;
}

struct TLifetimeSoundnessReason final {
    explicit constexpr TLifetimeSoundnessReason(std::string_view) {}
};

// A string_view-like class, *implicitly* convertible to TString.
// Used for try_emplace-ing a string_view into a HashMap<TString, ...>
class TStringViewConvertibleToString final {
public:
    TStringViewConvertibleToString(const TString&) = delete;
    TStringViewConvertibleToString(const std::string&) = delete;

    TStringViewConvertibleToString(const TString& data, TLifetimeSoundnessReason) : Data_{data} {}
    TStringViewConvertibleToString(const std::string& data, TLifetimeSoundnessReason) : Data_{data} {}
    TStringViewConvertibleToString(std::string_view data, TLifetimeSoundnessReason) : Data_{data} {}

    constexpr operator TStringBuf() const {
        return Data_;
    }

    operator TString() const {
        return TString{Data_};
    }
private:
    TStringBuf Data_;
};

struct SymbolizedProfileData final {
    std::vector<std::string_view> AllFunctions;

    // Indexes into AllFunctions
    std::vector<ui64> KernelFunctions;

    // Indexes into AllFunctions
    absl::flat_hash_map<ui64, TSmallVector<ui64>> FramesByLocationId;
};

SymbolizedProfileData SymbolizeProfile(
    absl::flat_hash_map<TString, TCachingGSYMSymbolizer>& symbolizers,
    const NPerforator::NProto::NPProf::ProfileLight& profile
) {
    absl::flat_hash_map<ui64, const NPerforator::NProto::NPProf::Function*> functionByIdMap;
    for (const auto& function : profile.Getfunction()) {
        functionByIdMap[function.id()] = &function;
    }

    absl::flat_hash_map<ui64, const NPerforator::NProto::NPProf::Mapping*> mappingByIdMap;
    for (const auto& mapping : profile.Getmapping()) {
        mappingByIdMap[mapping.id()] = &mapping;
    }

    absl::flat_hash_map<ui64, const NPerforator::NProto::NPProf::Location*> locationByIdMap;
    for (const auto& location : profile.Getlocation()) {
        locationByIdMap[location.id()] = &location;
    }

    SymbolizedProfileData result{};
    auto& allFunctions = result.AllFunctions;
    auto& kernelFunctions = result.KernelFunctions;
    auto& framesByLocationId = result.FramesByLocationId;

    allFunctions.reserve(profile.locationSize() * 2);
    framesByLocationId.reserve(profile.locationSize());

    // Every string, for which the views in this map are stored, outlives the map:
    // * some strings are static/constexpr
    // * some strings belong to the profile
    // * some strings belong to the TServicePerfTopAggregator (i.e. symbolization cache)
    absl::flat_hash_map<std::string_view, ui64> symbolizedFunctionsMapping;
    symbolizedFunctionsMapping.reserve(profile.locationSize() * 2);

    for (const auto& location : profile.Getlocation()) {
        TSmallVector<TStringViewConvertibleToString> symbolized{};
        if (location.lineSize() > 0) {
            for (const auto& line : location.Getline()) {
                const auto functionId = line.function_id();
                if (functionId == 0) {
                    symbolized.emplace_back(kUnknownFunction, TLifetimeSoundnessReason{"kUnknownFunction is static"});
                    continue;
                }
                const auto& function = *functionByIdMap.at(functionId);
                symbolized.emplace_back(
                    profile.string_table(function.name()),
                    TLifetimeSoundnessReason{"profile outlives everything in this function"}
                );
            }
        } else {
            [&symbolized, &location, &mappingByIdMap, &profile, &symbolizers]() {
                const auto mappingId = location.mapping_id();
                if (mappingId == 0) {
                    symbolized.emplace_back(kUnknownMapping, TLifetimeSoundnessReason{"kUnknownMapping is static"});
                    return;
                }
                const auto& mapping = *mappingByIdMap.at(mappingId);

                if (mapping.build_id() == 0) {
                    symbolized.emplace_back(kUnknownBuildId, TLifetimeSoundnessReason{"kUnknownBuildId is static"});
                    return;
                }
                const auto& buildId = profile.string_table(mapping.build_id());

                auto symbolizerIt = symbolizers.find(buildId);
                if (symbolizerIt == symbolizers.end()) {
                    symbolized.emplace_back(kNoGSYMLocation, TLifetimeSoundnessReason{"kNoGSYMLocation is static"});
                    return;
                }
                auto& symbolizer = symbolizerIt->second;

                const auto address = location.address() + mapping.file_offset() - mapping.memory_start();
                const auto& symbolizationResult = symbolizer.Symbolize(address);

                if (symbolizationResult.empty()) {
                    symbolized.emplace_back(kUnknownNoFrames, TLifetimeSoundnessReason{"kUnknownNoFrames is static"});
                    return;
                }

                for (const auto& frame : symbolizationResult) {
                    symbolized.emplace_back(
                        frame,
                        TLifetimeSoundnessReason{"symbolizationResult is cached in symbolizer, thus its lifetime is tied to the aggregator"}
                    );
                }
            }();
        }

        const auto isKernelLocation = [&mappingByIdMap, &profile] (ui64 mappingId) {
            if (mappingId == 0) {
                return false;
            }

            const auto& mapping = *mappingByIdMap.at(mappingId);
            return profile.string_table(mapping.filename()) == kKernelSpecialMapping;
        }(location.mapping_id());

        auto& frames = framesByLocationId[location.id()];
        for (const auto& function : symbolized) {
            const auto [it, inserted] = symbolizedFunctionsMapping.try_emplace(function, allFunctions.size());
            if (inserted) {
                allFunctions.push_back(function);
            }
            const auto functionIdx = it->second;

            frames.push_back(functionIdx);

            if (isKernelLocation) {
                kernelFunctions.push_back(functionIdx);
            }
        }
    }

    return result;
}

}

TCachingGSYMSymbolizer::TCachingGSYMSymbolizer(std::string_view gsymPath) : Symbolizer_{gsymPath} {
    SymbolizationCache_.reserve(128 * 1024);
}

const std::vector<std::string>& TCachingGSYMSymbolizer::Symbolize(ui64 addr) {
    auto [it, inserted] = SymbolizationCache_.try_emplace(addr);
    if (inserted) {
        auto& frames = it->second;

        auto symbolizationResult = Symbolizer_.Symbolize(addr);
        frames.reserve(symbolizationResult.size());
        for (auto& frame : symbolizationResult) {
            frames.push_back(std::move(frame.FunctionName));
        }
    }

    return it->second;
}

void TCachingGSYMSymbolizer::PruneCaches() {
    SymbolizationCache_.clear();
}

TServicePerfTopAggregator::TServicePerfTopAggregator() {}

void TServicePerfTopAggregator::InitializeSymbolizer(
    TArrayRef<const char> buildId,
    TArrayRef<const char> gsymPath
) {
    Symbolizers_.emplace(
        TString{buildId.data(), buildId.size()},
        std::string_view{gsymPath.data(), gsymPath.size()}
    );
}

void TServicePerfTopAggregator::AddProfile(TArrayRef<const char> service, TArrayRef<const char> profileBytes) {
    if (profileBytes.data() == nullptr || profileBytes.size() == 0) {
        return;
    }

    auto& profile = ProfileForMemoryReuse_;
    if (!profile.ParseFromString(std::string_view{profileBytes.data(), profileBytes.size()})) {
        return;
    }

    AddProfile(service, profile);
}

void TServicePerfTopAggregator::AddProfile(TArrayRef<const char>, const NPerforator::NProto::NPProf::ProfileLight& profile) {
    MaybePruneCaches();

    const auto symbolizedProfileData = SymbolizeProfile(Symbolizers_, profile);
    const auto& allFunctions = symbolizedProfileData.AllFunctions;
    const auto& kernelFunctions = symbolizedProfileData.KernelFunctions;
    const auto& framesByLocationId = symbolizedProfileData.FramesByLocationId;

    std::vector<ui128> cumulativeCyclesCountByFunctionId;
    cumulativeCyclesCountByFunctionId.assign(allFunctions.size(), 0);

    std::vector<ui64> lastSampleIdxForFunction;
    ui64 currentSampleIdx = 0;
    lastSampleIdxForFunction.assign(allFunctions.size(), currentSampleIdx);

    for (const auto& sample : profile.Getsample()) {
        ++currentSampleIdx;

        const auto value = GetCpuCyclesValue(profile, sample);
        TotalCycles_ += value;

        {
            // We must only account every unique function once here, otherwise
            // recursive and/or UNKNOWN functions get wrong cumulative values:
            // imagine a stack "A -> A -> B" with 5 cycles spent in it,
            // if we increment every function present by 5, we will get 10 for A,
            // which is obviously wrong.
            //
            // We implement the "unique functions in a sample" by keeping track
            // for each function when it was last encountered, this way we can avoid
            // sorting/hashing functionIds within the sample, which is noticeably faster.
            for (const auto& locationId : sample.Getlocation_id()) {
                const auto it = framesByLocationId.find(locationId);
                if (it == framesByLocationId.end()) {
                    continue;
                }
                for (const auto functionId : it->second) {
                    if (lastSampleIdxForFunction[functionId] != currentSampleIdx) {
                        cumulativeCyclesCountByFunctionId[functionId] += value;
                        lastSampleIdxForFunction[functionId] = currentSampleIdx;
                    }
                }
            }
        }

        if (sample.location_idSize() == 0) {
            continue;
        }

        const auto leafFrameIt = framesByLocationId.find(sample.location_id(0));
        if (leafFrameIt == framesByLocationId.end()) {
            continue;
        }
        const auto& leafFrame = leafFrameIt->second;

        if (!leafFrame.empty()) {
            CyclesByFunction_.try_emplace<TStringViewConvertibleToString>(
                TStringViewConvertibleToString{
                    allFunctions[leafFrame.back()],
                    TLifetimeSoundnessReason{"string_view-s in allFunctions are valid by construction, see SymbolizeProfile implementation"}
                },
                0
            ).first->second.SelfCycles += value;
        }
    }

    for (std::size_t i = 0; i < cumulativeCyclesCountByFunctionId.size(); ++i) {
        const auto value = cumulativeCyclesCountByFunctionId[i];
        if (value != 0) {
            CyclesByFunction_.try_emplace<TStringViewConvertibleToString>(
                TStringViewConvertibleToString{
                    allFunctions[i],
                    TLifetimeSoundnessReason{"string_view-s in allFunctions are valid by construction, see SymbolizeProfile implementation"}
                },
                0
            ).first->second.CumulativeCycles += value;
        }
    }

    for (const auto kernelFunctionIdx : kernelFunctions) {
        KernelFunctions_.emplace<TStringViewConvertibleToString>(TStringViewConvertibleToString{
            allFunctions[kernelFunctionIdx],
            TLifetimeSoundnessReason{"string_view-s in allFunctions are valid by construction, see SymbolizeProfile implementation"}
        });
    }

    ++TotalProfiles_;
}

void TServicePerfTopAggregator::MergeAggregator(const TServicePerfTopAggregator& other) {
    for (const auto& [k, v] : other.CyclesByFunction_) {
        auto& cyclesCount = CyclesByFunction_[k];
        cyclesCount.SelfCycles += v.SelfCycles;
        cyclesCount.CumulativeCycles += v.CumulativeCycles;
    }

    TotalCycles_ += other.TotalCycles_;
    TotalProfiles_ += other.TotalProfiles_;
}

TServicePerfTopAggregator::PerfTop TServicePerfTopAggregator::ExtractEntries() {
    const auto sortAndDemangle = [this](const auto& valueSelector) {
        std::vector<std::pair<TString, ui128>> total;
        total.reserve(CyclesByFunction_.size());
        for (const auto& [k, v] : CyclesByFunction_) {
            total.emplace_back(
                std::piecewise_construct,
                std::forward_as_tuple(k),
                std::forward_as_tuple(valueSelector(v))
            );
        }
        std::sort(total.begin(), total.end(), [](const auto& lhs, const auto& rhs) {
            return lhs.second > rhs.second;
        });

        if (total.size() > kMaxEntriesToPrint) {
            total.resize(kMaxEntriesToPrint);
        }

        for (auto& [name, _] : total) {
            const auto isKernelFunction = KernelFunctions_.contains(name);
            name = NPerforator::NSymbolize::CleanupFunctionName(
                NPerforator::NSymbolize::DemangleFunctionName(name)
            );

            if (isKernelFunction) {
                name = "[kernel] " + name;
            }
        }

        return total;
    };

    auto selfCycles = sortAndDemangle([](const auto& cyclesCount) { return cyclesCount.SelfCycles; });
    auto cumulativeCycles = sortAndDemangle([](const auto& cyclesCount) { return cyclesCount.CumulativeCycles; });

    struct CyclesCount final {
        ui128 SelfCycles = 0;
        ui128 CumulativeCycles = 0;
    };
    absl::flat_hash_map<TStringBuf, CyclesCount> cyclesCount;
    for (const auto& [name, cycles] : selfCycles) {
        cyclesCount[name].SelfCycles += cycles;
    }
    for (const auto& [name, cycles] : cumulativeCycles) {
        cyclesCount[name].CumulativeCycles += cycles;
    }

    std::vector<Function> functions;
    functions.reserve(cyclesCount.size());
    for (const auto& [name, cycles] : cyclesCount) {
        functions.push_back(Function{
            .Name = TString{name},
            .SelfCycles = cycles.SelfCycles,
            .CumulativeCycles = cycles.CumulativeCycles,
        });
    }

    return PerfTop{
        .Functions = std::move(functions),
        .TotalCycles = TotalCycles_
    };
}

void TServicePerfTopAggregator::MaybePruneCaches() {
    if (TotalProfiles_ % 100 == 0) {
        std::size_t totalSymbolizationCacheSize = 0;
        for (const auto& [_, symbolizer]: Symbolizers_) {
            totalSymbolizationCacheSize += symbolizer.GetCacheSize();
        }

        if (totalSymbolizationCacheSize > kMaxSymbolizationCacheTotalSize) {
            for (auto& [_, symbolizer] : Symbolizers_) {
                symbolizer.PruneCaches();
            }
        }
    }
}

}
