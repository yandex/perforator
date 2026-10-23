#include "perf_top_aggregator.h"

#include <perforator/lib/demangle/demangle.h>
#include <perforator/lib/profile/profile.h>

#include <library/cpp/yt/compact_containers/compact_vector.h>

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
    std::vector<TFunctionLocation> AllFunctions;

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
    auto& framesByFrameId = result.FramesByFrameId;

    const i32 frameCount = StackFrameCount(profile);
    framesByFrameId.resize(frameCount);
    allFunctions.reserve(frameCount * 2);

    absl::flat_hash_map<TFunctionLocation, ui64, TFunctionLocationHash> symbolizedFunctionsMapping;
    symbolizedFunctionsMapping.reserve(frameCount * 2);

    for (NProfile::TStackFrame frame : profile.StackFrames()) {
        const i32 frameIdx = frame.GetIndex().GetInternalIndex();
        auto& frames = framesByFrameId[frameIdx];

        TSmallVector<TFunctionLocation> symbolized{};
        const NProfile::TBinary binary = frame.GetBinary();
        const bool hasBinary = binary.GetIndex() != NProfile::TBinaryId::Zero();
        const TString buildId = hasBinary ? TString{binary.GetBuildId().View()} : TString{};
        const bool isKernel = hasBinary && binary.GetPath().View() == kKernelSpecialMapping;
        const auto unknown = [&](std::string_view name) {
            symbolized.push_back(TFunctionLocation{.Name = TString{name}});
        };

        const NProfile::TInlineChain inlineChain = frame.GetInlineChain();
        if (inlineChain.GetIndex() != NProfile::TInlineChainId::Zero()) {
            // Frame is already symbolized: use pre-existing inline chain.
            if (std::ranges::empty(inlineChain.GetLines())) {
                unknown(kUnknownFunction);
            } else {
                // Inline chains are stored innermost-first (leaf-to-root); see flamegraph/build.cpp.
                for (NProfile::TSourceLine sourceLine : inlineChain.GetLines()) {
                    NProfile::TFunction func = sourceLine.GetFunction();
                    // A zero function reference can still carry a source position.
                    const bool hasFunction = func.GetIndex() != NProfile::TFunctionId::Zero();
                    symbolized.push_back(TFunctionLocation{
                        .Name = hasFunction ? TString{func.GetName().View()} : TString{kUnknownFunction},
                        .SourceFile = hasFunction ? TString{func.GetFileName().View()} : TString{},
                        .SourceLine = sourceLine.GetLine(),
                        .SourceColumn = sourceLine.GetColumn(),
                    });
                }
            }
        } else {
            // Frame not symbolized: look up GSYM by buildId + address.
            [&]() {
                if (!hasBinary) {
                    unknown(kUnknownMapping);
                    return;
                }
                if (buildId.empty()) {
                    unknown(kUnknownBuildId);
                    return;
                }
                auto symbolizerIt = symbolizers.find(buildId);
                if (symbolizerIt == symbolizers.end()) {
                    unknown(kNoGSYMLocation);
                    return;
                }

                // NProfile stores address in binary address space directly —
                // no file_offset/memory_start arithmetic needed.
                const ui64 addr = frame.GetAddress();
                const auto& symbolizationResult = symbolizerIt->second.Symbolize(addr);

                if (symbolizationResult.empty()) {
                    unknown(kUnknownNoFrames);
                    return;
                }

                for (const auto& location : symbolizationResult) {
                    symbolized.push_back(location);
                }
            }();
        }

        for (auto& location : symbolized) {
            location.BuildId = buildId;
            location.IsKernel = isKernel;
            const auto [it, inserted] = symbolizedFunctionsMapping.try_emplace(
                location, allFunctions.size()
            );
            if (inserted) {
                allFunctions.push_back(std::move(location));
            }
            const auto functionIdx = it->second;
            frames.push_back(functionIdx);
        }
    }

    return result;
}

} // namespace


TCachingGSYMSymbolizer::TCachingGSYMSymbolizer(std::string_view gsymPath) : Symbolizer_{gsymPath} {
    SymbolizationCache_.reserve(128 * 1024);
}

const std::vector<TFunctionLocation>& TCachingGSYMSymbolizer::Symbolize(ui64 addr) {
    auto [it, inserted] = SymbolizationCache_.try_emplace(addr);
    if (inserted) {
        auto& frames = it->second;

        auto symbolizationResult = Symbolizer_.Symbolize(addr);
        frames.reserve(symbolizationResult.size());
        for (auto& frame : symbolizationResult) {
            frames.push_back(TFunctionLocation{
                .Name = TString{std::move(frame.FunctionName)},
                .SourceFile = TString{std::move(frame.FileName)},
                .SourceLine = frame.Line,
                .SourceColumn = frame.Column,
            });
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
    const auto& framesByFrameId = symbolizedData.FramesByFrameId;

    std::vector<TFunctionStats> cyclesCountByFunctionId(allFunctions.size());
    std::vector<bool> hasSelfSamples(allFunctions.size(), false);

    std::vector<ui64> lastSampleIdxForFunction;
    ui64 currentSampleIdx = 0;
    lastSampleIdxForFunction.assign(allFunctions.size(), currentSampleIdx);

    for (NProfile::TSample sample : profile.Samples()) {
        ++currentSampleIdx;

        const ui64 value = sample.GetValues()[cpuValueIdx].GetValue();
        TotalCycles_ += value;

        NProfile::TSampleKey sampleKey = sample.GetKey();

        {
            // Count each source location once per sample. Different lines of a
            // recursive function are separate contributions to the function MV.
            for (NProfile::TStack stack : sampleKey.GetStacks()) {
                for (NProfile::TStackFrame frame : stack.GetFrames()) {
                    const i32 frameId = frame.GetIndex().GetInternalIndex();
                    if (frameId < 0 || static_cast<std::size_t>(frameId) >= framesByFrameId.size()) {
                        continue;
                    }
                    for (const ui64 functionIdx : framesByFrameId[frameId]) {
                        if (lastSampleIdxForFunction[functionIdx] != currentSampleIdx) {
                            cyclesCountByFunctionId[functionIdx].CumulativeCycles += value;
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
        cyclesCountByFunctionId[leafFunctionIdx].SelfCycles += value;
        hasSelfSamples[leafFunctionIdx] = true;
    }

    for (std::size_t i = 0; i < cyclesCountByFunctionId.size(); ++i) {
        const auto& cycles = cyclesCountByFunctionId[i];
        // Keep zero-valued leaf rows, but hash each full location only once per profile.
        if (hasSelfSamples[i] || cycles.CumulativeCycles != 0) {
            auto& count = CyclesByFunction_[allFunctions[i]];
            count.SelfCycles += cycles.SelfCycles;
            count.CumulativeCycles += cycles.CumulativeCycles;
        }
    }

    ++TotalProfiles_;
}

void TPerfTopAggregator::MergeAggregator(const TPerfTopAggregator& other) {
    for (const auto& [k, v] : other.CyclesByFunction_) {
        auto& cyclesCount = CyclesByFunction_[k];
        cyclesCount.SelfCycles += v.SelfCycles;
        cyclesCount.CumulativeCycles += v.CumulativeCycles;
    }

    TotalCycles_ += other.TotalCycles_;
    TotalProfiles_ += other.TotalProfiles_;
}

TPerfTopAggregator::PerfTop TPerfTopAggregator::ExtractEntries() {
    absl::flat_hash_map<TFunctionLocation, TFunctionStats, TFunctionLocationHash> cyclesCount;
    cyclesCount.reserve(CyclesByFunction_.size());
    // Raw names live in the unchanged accumulator throughout extraction.
    absl::flat_hash_map<TStringBuf, TString> demangledNames;
    for (const auto& [location, cycles] : CyclesByFunction_) {
        auto [nameIt, inserted] = demangledNames.try_emplace(location.Name);
        if (inserted) {
            nameIt->second = NDemangle::Demangle(location.Name);
        }
        auto demangled = location;
        demangled.Name = nameIt->second;
        if (location.IsKernel) {
            demangled.Name = "[kernel] " + demangled.Name;
        }
        auto& count = cyclesCount[std::move(demangled)];
        count.SelfCycles += cycles.SelfCycles;
        count.CumulativeCycles += cycles.CumulativeCycles;
    }

    std::vector<Function> functions;
    functions.reserve(cyclesCount.size());
    for (const auto& [location, cycles] : cyclesCount) {
        functions.push_back(Function{
            .Name = location.Name,
            .BuildId = location.BuildId,
            .SourceFile = location.SourceFile,
            .SourceLine = location.SourceLine,
            .SourceColumn = location.SourceColumn,
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
