#include "perf_top_aggregator.h"

#include <perforator/lib/demangle/demangle.h>
#include <perforator/lib/profile/profile.h>

#include <library/cpp/yt/compact_containers/compact_vector.h>

#include <algorithm>
#include <ranges>
#include <string_view>

namespace NPerforator::NClusterTop {

namespace {

template<typename T>
using TSmallVector = NYT::TCompactVector<T, 4>;

constexpr std::string_view kCPUCyclesType = "cpu";
constexpr std::string_view kCPUCyclesUnit = "cycles";

constexpr std::string_view kUnknownFunction = "<UNKNOWN (No function)>";
constexpr std::string_view kUnknownMapping = "<UNKNOWN (No mapping)>";
constexpr std::string_view kUnknownBuildId = "<UNKNOWN (No buildId)>";
constexpr std::string_view kUnknownNoFrames = "<UNKNOWN (No frames)>";
constexpr std::string_view kNoGSYMLocation = "<UNKNOWN (No GSYM)>";

constexpr std::string_view kKernelSpecialMapping = "[kernel]";

constexpr std::size_t kMaxSymbolizationCacheTotalSize = 256 * 1024;

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

// Returns the index into sample value types that corresponds to cpu/cycles, or -1.
i32 FindCpuCyclesValueTypeIndex(const NProfile::TProfile& profile) {
    i32 i = 0;
    for (NProfile::TValueType valueType : profile.ValueTypes()) {
        if (valueType.GetType().View() == kCPUCyclesType && valueType.GetUnit().View() == kCPUCyclesUnit) {
            return i;
        }
        ++i;
    }
    return -1;
}

i32 StackFrameCount(const NProfile::TProfile& profile) {
    return static_cast<i32>(std::ranges::size(profile.StackFrames()));
}

struct SymbolizedProfileData final {
    std::vector<std::string_view> AllFunctions;

    // Indices into AllFunctions that belong to kernel frames.
    std::vector<ui64> KernelFunctions;

    // Per stack frame (indexed by TStackFrameId internal index, i.e. position
    // in the stack_frames arrays): list of indices into AllFunctions.
    // Order within each entry: [0] = most-inlined (leaf), [back] = outermost.
    std::vector<TSmallVector<ui64>> FramesByFrameId;
};

SymbolizedProfileData SymbolizeProfile(
    absl::flat_hash_map<TString, TCachingGSYMSymbolizer>& symbolizers,
    const NProfile::TProfile& profile
) {
    SymbolizedProfileData result{};
    auto& allFunctions = result.AllFunctions;
    auto& kernelFunctions = result.KernelFunctions;
    auto& framesByFrameId = result.FramesByFrameId;

    const i32 frameCount = StackFrameCount(profile);
    framesByFrameId.resize(frameCount);
    allFunctions.reserve(frameCount * 2);

    // All string_view-s stored here either:
    // * point to compile-time constants (static storage duration), or
    // * point into profile strtab (alive for the duration of AddProfile), or
    // * point into TCachingGSYMSymbolizer caches (lifetime tied to the aggregator).
    absl::flat_hash_map<std::string_view, ui64> symbolizedFunctionsMapping;
    symbolizedFunctionsMapping.reserve(frameCount * 2);

    for (NProfile::TStackFrame frame : profile.StackFrames()) {
        const i32 frameIdx = frame.GetIndex().GetInternalIndex();
        auto& frames = framesByFrameId[frameIdx];

        TSmallVector<TStringViewConvertibleToString> symbolized{};

        const NProfile::TInlineChain inlineChain = frame.GetInlineChain();
        if (inlineChain.GetIndex() != NProfile::TInlineChainId::Zero()) {
            // Frame is already symbolized: use pre-existing inline chain.
            if (std::ranges::empty(inlineChain.GetLines())) {
                symbolized.emplace_back(
                    kUnknownFunction,
                    TLifetimeSoundnessReason{"kUnknownFunction is static"}
                );
            } else {
                // Inline chains are stored innermost-first (leaf-to-root); see flamegraph/build.cpp.
                for (NProfile::TSourceLine sourceLine : inlineChain.GetLines()) {
                    NProfile::TFunction func = sourceLine.GetFunction();
                    if (func.GetIndex() == NProfile::TFunctionId::Zero()) {
                        // Original pprof Line.function_id == 0 (no function ref).
                        // ConvertFromPProf maps it to the sentinel zero function;
                        // preserve the old "<UNKNOWN (No function)>" behavior.
                        symbolized.emplace_back(
                            kUnknownFunction,
                            TLifetimeSoundnessReason{"kUnknownFunction is static"}
                        );
                    } else {
                        symbolized.emplace_back(
                            func.GetName().View(),
                            TLifetimeSoundnessReason{"string lives in rawProfile strtab"}
                        );
                    }
                }
            }
        } else {
            // Frame not symbolized: look up GSYM by buildId + address.
            [&]() {
                const NProfile::TBinary binary = frame.GetBinary();
                if (binary.GetIndex() == NProfile::TBinaryId::Zero()) {
                    symbolized.emplace_back(
                        kUnknownMapping,
                        TLifetimeSoundnessReason{"kUnknownMapping is static"}
                    );
                    return;
                }
                const TStringBuf buildId = binary.GetBuildId().View();
                if (buildId.empty()) {
                    symbolized.emplace_back(
                        kUnknownBuildId,
                        TLifetimeSoundnessReason{"kUnknownBuildId is static"}
                    );
                    return;
                }
                auto symbolizerIt = symbolizers.find(TString{buildId});
                if (symbolizerIt == symbolizers.end()) {
                    symbolized.emplace_back(
                        kNoGSYMLocation,
                        TLifetimeSoundnessReason{"kNoGSYMLocation is static"}
                    );
                    return;
                }

                // NProfile stores address in binary address space directly —
                // no file_offset/memory_start arithmetic needed.
                const ui64 addr = frame.GetAddress();
                const auto& symbolizationResult = symbolizerIt->second.Symbolize(addr);

                if (symbolizationResult.empty()) {
                    symbolized.emplace_back(
                        kUnknownNoFrames,
                        TLifetimeSoundnessReason{"kUnknownNoFrames is static"}
                    );
                    return;
                }

                for (const auto& funcName : symbolizationResult) {
                    symbolized.emplace_back(
                        funcName,
                        TLifetimeSoundnessReason{"result is cached in TCachingGSYMSymbolizer"}
                    );
                }
            }();
        }

        const bool isKernel = [&]() {
            const NProfile::TBinary binary = frame.GetBinary();
            if (binary.GetIndex() == NProfile::TBinaryId::Zero()) {
                return false;
            }
            return binary.GetPath().View() == kKernelSpecialMapping;
        }();

        for (const auto& funcName : symbolized) {
            const auto [it, inserted] = symbolizedFunctionsMapping.try_emplace(
                funcName, allFunctions.size()
            );
            if (inserted) {
                allFunctions.push_back(funcName);
            }
            const auto functionIdx = it->second;
            frames.push_back(functionIdx);
            if (isKernel) {
                kernelFunctions.push_back(functionIdx);
            }
        }
    }

    return result;
}

} // namespace


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

TPerfTopAggregator::TPerfTopAggregator() {}

void TPerfTopAggregator::InitializeSymbolizer(
    TArrayRef<const char> buildId,
    TArrayRef<const char> gsymPath
) {
    Symbolizers_.emplace(
        TString{buildId.data(), buildId.size()},
        std::string_view{gsymPath.data(), gsymPath.size()}
    );
}

void TPerfTopAggregator::AddProfile(
    const NPerforator::NProto::NProfile::Profile& rawProfile
) {
    MaybePruneCaches();

    NProfile::TProfile profile{&rawProfile};

    const i32 cpuValueIdx = FindCpuCyclesValueTypeIndex(profile);
    if (cpuValueIdx < 0) {
        return;
    }

    const auto symbolizedData = SymbolizeProfile(Symbolizers_, profile);
    const auto& allFunctions = symbolizedData.AllFunctions;
    const auto& kernelFunctions = symbolizedData.KernelFunctions;
    const auto& framesByFrameId = symbolizedData.FramesByFrameId;

    std::vector<ui128> cumulativeCyclesCountByFunctionId;
    cumulativeCyclesCountByFunctionId.assign(allFunctions.size(), 0);

    std::vector<ui64> lastSampleIdxForFunction;
    ui64 currentSampleIdx = 0;
    lastSampleIdxForFunction.assign(allFunctions.size(), currentSampleIdx);

    for (NProfile::TSample sample : profile.Samples()) {
        ++currentSampleIdx;

        const ui64 value = sample.GetValues()[cpuValueIdx].GetValue();
        TotalCycles_ += value;

        NProfile::TSampleKey sampleKey = sample.GetKey();

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
            for (NProfile::TStack stack : sampleKey.GetStacks()) {
                for (NProfile::TStackFrame frame : stack.GetFrames()) {
                    const i32 frameId = frame.GetIndex().GetInternalIndex();
                    if (frameId < 0 || static_cast<std::size_t>(frameId) >= framesByFrameId.size()) {
                        continue;
                    }
                    for (const ui64 functionIdx : framesByFrameId[frameId]) {
                        if (lastSampleIdxForFunction[functionIdx] != currentSampleIdx) {
                            cumulativeCyclesCountByFunctionId[functionIdx] += value;
                            lastSampleIdxForFunction[functionIdx] = currentSampleIdx;
                        }
                    }
                }
            }
        }

        // Self cycles belong to the innermost inline function of the top (leaf)
        // frame of the first stack. Inline chains are ordered leaf-to-root.
        auto stacks = sampleKey.GetStacks();
        const auto stacksIt = std::ranges::begin(stacks);
        if (stacksIt == std::ranges::end(stacks)) {
            continue;
        }
        const NProfile::TStackFrame leafFrame = (*stacksIt).GetTopFrame();
        const i32 leafFrameId = leafFrame.GetIndex().GetInternalIndex();
        if (leafFrameId < 0 || static_cast<std::size_t>(leafFrameId) >= framesByFrameId.size()) {
            continue;
        }
        const auto& leafFrameFunctions = framesByFrameId[leafFrameId];
        if (leafFrameFunctions.empty()) {
            continue;
        }

        const ui64 leafFunctionIdx = leafFrameFunctions.front();
        CyclesByFunction_.try_emplace<TStringViewConvertibleToString>(
            TStringViewConvertibleToString{
                allFunctions[leafFunctionIdx],
                TLifetimeSoundnessReason{"string_view-s in allFunctions are valid by construction, see SymbolizeProfile"}
            },
            0
        ).first->second.SelfCycles += value;
    }

    for (std::size_t i = 0; i < cumulativeCyclesCountByFunctionId.size(); ++i) {
        const auto value = cumulativeCyclesCountByFunctionId[i];
        if (value != 0) {
            CyclesByFunction_.try_emplace<TStringViewConvertibleToString>(
                TStringViewConvertibleToString{
                    allFunctions[i],
                    TLifetimeSoundnessReason{"string_view-s in allFunctions are valid by construction, see SymbolizeProfile"}
                },
                0
            ).first->second.CumulativeCycles += value;
        }
    }

    for (const ui64 kernelFunctionIdx : kernelFunctions) {
        KernelFunctions_.emplace<TStringViewConvertibleToString>(TStringViewConvertibleToString{
            allFunctions[kernelFunctionIdx],
            TLifetimeSoundnessReason{"string_view-s in allFunctions are valid by construction, see SymbolizeProfile"}
        });
    }

    ++TotalProfiles_;
}

void TPerfTopAggregator::MergeAggregator(const TPerfTopAggregator& other) {
    for (const auto& [k, v] : other.CyclesByFunction_) {
        auto& cyclesCount = CyclesByFunction_[k];
        cyclesCount.SelfCycles += v.SelfCycles;
        cyclesCount.CumulativeCycles += v.CumulativeCycles;
    }

    KernelFunctions_.insert(other.KernelFunctions_.begin(), other.KernelFunctions_.end());

    TotalCycles_ += other.TotalCycles_;
    TotalProfiles_ += other.TotalProfiles_;
}

TPerfTopAggregator::PerfTop TPerfTopAggregator::ExtractEntries() {
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

        for (auto& [name, _] : total) {
            const auto isKernelFunction = KernelFunctions_.contains(name);
            name = NDemangle::Demangle(name);

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

void TPerfTopAggregator::MaybePruneCaches() {
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
