#include "profile_accumulator.h"

#include "signals.h"

#include <perforator/lib/profile/builder.h>
#include <perforator/lib/profile/well_known.h>
#include <perforator/proto/profile/profile.pb.h>

#include <library/cpp/int128/int128.h>

#include <util/generic/strbuf.h>
#include <util/generic/yexception.h>
#include <util/string/cast.h>

#include <array>
#include <cstdint>
#include <cstring>
#include <limits>
#include <optional>
#include <utility>

namespace NPerforator::NAgent::NNativePipeline {
namespace {

using namespace NPerforator::NProfile;
using TLowLevelProfileBuilder = NPerforator::NProfile::TProfileBuilder;
using NProto::NProfile::WellKnownLabel;

constexpr ui64 KernelAddressAdjustment = 0xffffffffffff0000ull;

ui64 ScalePerfCounterDelta(const perf_event_value& delta) {
    const ui64 counter = delta.counter;
    const ui64 enabled = delta.enabled;
    const ui64 running = delta.running;
    const ui128 product = ui128{counter} * enabled;
    // Match the Go consumer's overflow sentinel and preserve the value bits otherwise.
    if (GetHigh(product) >= running) {
        return std::numeric_limits<i64>::max();
    }
    return GetLow(product / running);
}

TStringBuf Comm(const std::uint8_t (&value)[16]) {
    const auto* begin = reinterpret_cast<const char*>(value);
    const auto* terminator = static_cast<const char*>(std::memchr(begin, '\0', sizeof(value)));
    return {begin, terminator ? static_cast<size_t>(terminator - begin) : sizeof(value)};
}

size_t WorkloadCgroupCount(const TRawSample& sample) {
    size_t count = sample.Cgroups().Count();
    if (sample.Header().parent_cgroup == END_OF_CGROUP_LIST && count > 0 && count < PARENT_CGROUP_MAX_LEVELS) {
        --count;
    }
    return count;
}

// The returned view borrows variable or formattedNumber.
std::optional<TStringBuf> FormatTLSValue(const thread_local_variable_collect_result& variable, TString& formattedNumber) {
    switch (variable.type) {
        case THREAD_LOCAL_UINT64_TYPE:
            formattedNumber = ToString(variable.value.number);
            return TStringBuf{formattedNumber};
        case THREAD_LOCAL_STRING_TYPE:
            if (variable.value.string.len > sizeof(variable.value.string.string)) {
                return std::nullopt;
            }
            return TStringBuf{variable.value.string.string, variable.value.string.len};
        default:
            return std::nullopt;
    }
}

const TPerfEventInfo* FindPerfEvent(const TStateSnapshot& state, ui64 id) {
    Y_ENSURE(state.Global.PerfEvents, "perf sample requires an event snapshot");
    const auto& events = state.Global.PerfEvents->Events;
    if (auto it = events.find(id); it != events.end()) {
        return &it->second;
    }
    if (state.Global.Targets) {
        const auto& custom = state.Global.Targets->CustomPerfEvents;
        if (auto it = custom.find(id); it != custom.end()) {
            return &it->second;
        }
    }
    return nullptr;
}

TRuntimeFrameBatch ResolveRuntimeFrames(const TRawSample& sample, IRuntimeResolver& resolver) {
    const auto nativeStack = sample.UserStack();
    TVector<ui64> nativeAddresses;
    nativeAddresses.reserve(nativeStack.Count());
    bool hasNonzeroNativeAddresses = false;
    for (size_t i = 0; i < nativeStack.Count(); ++i) {
        const auto address = nativeStack.Read(i);
        nativeAddresses.push_back(address);
        hasNonzeroNativeAddresses |= address != 0;
    }

    const size_t languageCount = sample.LanguageSectionCount();
    if (!hasNonzeroNativeAddresses && languageCount == 0) {
        return {};
    }

    std::array<TPerforatorLanguageSection, PERFORATOR_LANGUAGE_COUNT> languageSections{};
    for (size_t i = 0; i < languageCount; ++i) {
        const auto& section = sample.LanguageSectionAt(i);
        languageSections[i] = {
            .Data = section.Element(0).data(),
            .Size = section.Count() * section.ElementSize(),
            .FrameSize = static_cast<ui16>(section.ElementSize()),
            .Language = static_cast<EPerforatorLanguage>(section.Language()),
        };
    }
    return resolver.ResolveRuntimeFrames({
        .Key = {.Pid = sample.Header().pid, .StartTime = sample.Header().starttime},
        .NativeStack = nativeAddresses.data(),
        .NativeStackSize = nativeAddresses.size(),
        .Languages = languageSections.data(),
        .LanguagesSize = languageCount,
    });
}

TSampleTimestamp NormalizeTimestamp(i64 timestampNanos) {
    constexpr i64 NanosPerSecond = 1'000'000'000;
    i64 seconds = timestampNanos / NanosPerSecond;
    i64 nanos = timestampNanos % NanosPerSecond;
    if (nanos < 0) {
        --seconds;
        nanos += NanosPerSecond;
    }
    return {.Seconds = seconds, .NanoSeconds = static_cast<ui32>(nanos)};
}

class TProfileStackWriter {
public:
    explicit TProfileStackWriter(TLowLevelProfileBuilder& builder)
        : Builder_{builder}
    {
    }

    bool AppendStack(
        TLowLevelProfileBuilder::TSimpleSampleKeyBuilder& sampleKeyBuilder,
        const TRuntimeFrameBatch& runtimeFrames,
        TRawSection<std::uint64_t> kernelStack,
        const TKernelSymbolState* kernelSymbols,
        const TProcessState* process
    );
    bool AppendLbrStack(
        TLowLevelProfileBuilder::TSimpleSampleKeyBuilder& sampleKeyBuilder,
        TRawSection<branch_record> branches,
        const TProcessState* process
    );

private:
    TStackFrameId AddUserFrame(TResolvedFrameView frame, const TProcessState* process);
    TStackFrameId AddKernelFrame(ui64 address, const TKernelSymbolState* symbols);
    TStackFrameId AddNativeFrame(ui64 address, const TProcessState* process);
    TStackFrameId AddRuntimeFrame(TResolvedFrameView frame);
    TStackFrameId AddFrame(
        ui64 address,
        TStringBuf path,
        TStringBuf buildId,
        bool hasSkewedAddresses,
        TInlineChainId inlineChainId
    );

    TLowLevelProfileBuilder& Builder_;
};

bool TProfileStackWriter::AppendStack(
    TLowLevelProfileBuilder::TSimpleSampleKeyBuilder& sampleKeyBuilder,
    const TRuntimeFrameBatch& runtimeFrames,
    TRawSection<std::uint64_t> kernelStack,
    const TKernelSymbolState* kernelSymbols,
    const TProcessState* process
)
{
    bool hasFrames = false;
    const auto unboundFrames = runtimeFrames.GetUnboundLanguageFrames();
    for (size_t i = 0; i < unboundFrames.size(); ++i) {
        const auto frame = unboundFrames[i];
        if (frame.HasData()) {
            sampleKeyBuilder.AddFrame(AddRuntimeFrame(frame));
            hasFrames = true;
        }
    }
    for (size_t i = 0; i < kernelStack.Count(); ++i) {
        const auto address = kernelStack.Read(i);
        if (address) {
            sampleKeyBuilder.AddFrame(AddKernelFrame(address, kernelSymbols));
            hasFrames = true;
        }
    }
    const auto userFrames = runtimeFrames.GetUserFrames();
    for (size_t i = 0; i < userFrames.size(); ++i) {
        const auto frame = userFrames[i];
        if (frame.HasData()) {
            sampleKeyBuilder.AddFrame(AddUserFrame(frame, process));
            hasFrames = true;
        }
    }
    return hasFrames;
}

bool TProfileStackWriter::AppendLbrStack(
    TLowLevelProfileBuilder::TSimpleSampleKeyBuilder& sampleKeyBuilder,
    TRawSection<branch_record> branches,
    const TProcessState* process
)
{
    bool hasFrames = false;
    for (size_t i = 0; i < branches.Count(); ++i) {
        const auto branch = branches.Read(i);
        if (branch.from == 0 || branch.to == 0) {
            break;
        }
        sampleKeyBuilder.AddFrame(AddNativeFrame(branch.from, process));
        sampleKeyBuilder.AddFrame(AddNativeFrame(branch.to, process));
        hasFrames = true;
    }
    return hasFrames;
}

TStackFrameId TProfileStackWriter::AddUserFrame(TResolvedFrameView frame, const TProcessState* process) {
    if (frame.IsNativeAddress()) {
        return AddNativeFrame(frame.Address(), process);
    }
    return AddRuntimeFrame(frame);
}

TStackFrameId TProfileStackWriter::AddKernelFrame(ui64 address, const TKernelSymbolState* symbols) {
    const auto symbol = symbols ? symbols->Resolve(address) : TStringBuf{};
    const ui64 profileAddress = !symbol.empty() && symbol != "unknown"
                                    ? KernelAddressAdjustment
                                    : address + KernelAddressAdjustment;
    auto inlineChainId = TInlineChainId::Zero();
    if (!symbol.empty()) {
        auto inlineChainBuilder = Builder_.AddInlineChain();
        auto functionId = Builder_.AddFunction()
            .SetName(symbol)
            .SetSystemName(symbol)
            .Finish();
        inlineChainBuilder.AddLine().SetFunction(functionId).Finish();
        inlineChainId = inlineChainBuilder.Finish();
    }
    return AddFrame(profileAddress, "[kernel]", {}, true, inlineChainId);
}

TStackFrameId TProfileStackWriter::AddNativeFrame(ui64 address, const TProcessState* process) {
    const auto* mapping = address && process ? process->ResolveMapping(address) : nullptr;
    if (mapping) {
        return AddFrame(address + mapping->Offset - mapping->Begin,
                        mapping->Path, mapping->BuildId, true, TInlineChainId::Zero());
    }
    return AddFrame(address, {}, {}, false, TInlineChainId::Zero());
}

TStackFrameId TProfileStackWriter::AddRuntimeFrame(TResolvedFrameView frame) {
    auto inlineChainId = TInlineChainId::Zero();
    if (frame.SymbolCount()) {
        auto inlineChainBuilder = Builder_.AddInlineChain();
        for (size_t i = 0; i < frame.SymbolCount(); ++i) {
            const auto symbol = frame.Symbol(i);
            const auto name = symbol.Name();
            auto functionBuilder = Builder_.AddFunction()
                .SetName(name)
                .SetSystemName(name)
                .SetFileName(symbol.FileName());
            if (symbol.StartLine()) {
                functionBuilder.SetStartLine(symbol.StartLine());
            }
            auto lineBuilder = inlineChainBuilder.AddLine().SetFunction(functionBuilder.Finish());
            if (symbol.Line()) {
                lineBuilder.SetLine(symbol.Line());
            }
            lineBuilder.Finish();
        }
        inlineChainId = inlineChainBuilder.Finish();
    }
    return AddFrame(frame.Address(), frame.MappingName(), {}, true, inlineChainId);
}

TStackFrameId TProfileStackWriter::AddFrame(
    ui64 address,
    TStringBuf path,
    TStringBuf buildId,
    bool hasSkewedAddresses,
    TInlineChainId inlineChainId
)
{
    TBinaryId binaryId = TBinaryId::Zero();
    if (!path.empty() || !buildId.empty() || hasSkewedAddresses) {
        binaryId = Builder_.AddBinary()
            .SetBuildId(buildId)
            .SetPath(path)
            .SetHasSkewedAddresses(hasSkewedAddresses)
            .Finish();
    }
    return Builder_.AddStackFrame()
        .SetBinary(binaryId)
        .SetAddress(address)
        .SetInlineChain(inlineChainId)
        .Finish();
}

} // namespace

TSampleAccumulator::TSampleAccumulator(
    NProfile::TProfileBuilder& builder,
    TMonotonicClockConverter& clock,
    TProfileOutput output,
    TProfileOptions options,
    bool emitPerSampleTimestamps
)
    : Output_{std::move(output)}
    , Options_{std::move(options)}
    , EmitPerSampleTimestamps_{emitPerSampleTimestamps}
    , Builder_{builder}
    , Clock_{clock}
{
    Y_ENSURE(Output_.Kind != EProfileOutput::LastBranchRecords ||
                 (!Output_.IncludeWallTime && Output_.UprobeSampleKind.empty()),
             "LBR output cannot contain wall values or uprobe configuration");
}

bool TSampleAccumulator::Accumulate(
    const TStateSnapshot& state,
    IRuntimeResolver& resolver,
    const TRawSample& sample
)
{
    Y_ENSURE(!Finished_, "sample accumulator has already been finished");
    Y_ENSURE(state.Global.Cgroups, "sample writing requires a cgroup snapshot");
    const auto& header = sample.Header();
    const auto values = BuildSampleValues(header, state);
    if (values.empty()) {
        return false;
    }
    CheckValueTypes(values);

    const auto process = state.FindProcess({header.pid, header.starttime});
    const auto signalName = header.sample_type == SAMPLE_TYPE_TRACEPOINT_SIGNAL_DELIVER
                                ? std::optional<TString>{GetSignalName(header.sample_config.sig)}
                                : std::nullopt;
    const auto timestampNanos = Clock_.MonotonicToUnixNanos(header.collection_time);
    const bool isFullStack = Output_.Kind == EProfileOutput::Event;
    const auto runtimeFrames = isFullStack ? ResolveRuntimeFrames(sample, resolver) : TRuntimeFrameBatch{};

    auto sampleKeyBuilder = Builder_.AddSimpleSampleKey();
    TProfileStackWriter stackWriter{Builder_};
    const bool hasFrames = isFullStack
                               ? stackWriter.AppendStack(
                                     sampleKeyBuilder,
                                     runtimeFrames,
                                     sample.KernelStack(),
                                     state.Global.KernelSymbols.get(),
                                     process.get())
                               : stackWriter.AppendLbrStack(sampleKeyBuilder, sample.Lbr(), process.get());
    InitializeValueTypes(values);
    if (!hasFrames) {
        FeedTimestamp(timestampNanos);
        return false;
    }

    AddCommonLabels(sampleKeyBuilder, state, process.get(), sample);
    if (isFullStack && Options_.EnableAbsoluteTimestampLabel) {
        sampleKeyBuilder.AddLabel(Builder_.AddNumericLabel(GetWellKnownLabelKey(WellKnownLabel::AbsoluteTimestamp), timestampNanos));
    }
    if (signalName) {
        sampleKeyBuilder.AddLabel(Builder_.AddStringLabel(GetWellKnownLabelKey(WellKnownLabel::SignalName), *signalName));
    }

    TSampleInfo output{
        .Key = sampleKeyBuilder.Finish(),
    };
    if (EmitPerSampleTimestamps_) {
        output.Timestamp = NormalizeTimestamp(timestampNanos);
    }
    for (size_t i = 0; i < values.size(); ++i) {
        output.Values.push_back({ValueTypeIds_[i], values[i].Value});
    }
    Builder_.AddSample(output);
    FeedTimestamp(timestampNanos);
    return true;
}

void TSampleAccumulator::Finish() {
    Y_ENSURE(!Finished_, "sample accumulator has already been finished");
    Finished_ = true;
    for (const auto& label : Options_.Labels) {
        Builder_.AddComment(label.Key + ":" + label.Value);
    }

    if (HasTimestampRange_) {
        auto& profileMetadata = Builder_.Metadata().GetProto();
        auto* profileTimestamp = profileMetadata.mutable_timestamp();
        const auto startTimestamp = NormalizeTimestamp(MinTimestampNanos_);
        profileTimestamp->set_seconds(startTimestamp.Seconds);
        profileTimestamp->set_nanos(startTimestamp.NanoSeconds);
        profileMetadata.set_duration_nanoseconds(MaxTimestampNanos_ - MinTimestampNanos_);
    }
    std::move(Builder_).Finish();
}

TVector<TSampleAccumulator::TSampleValue> TSampleAccumulator::BuildSampleValues(
    const record_sample_header& header,
    const TStateSnapshot& state
) const
{
    if (Output_.Kind != EProfileOutput::Event) {
        if (header.sample_type != SAMPLE_TYPE_PERF_EVENT) {
            return {};
        }
        return {{{"lbr", "stacks"}, ScalePerfCounterDelta(header.value)}};
    }

    TVector<TSampleValue> values;
    switch (header.sample_type) {
        case SAMPLE_TYPE_PERF_EVENT: {
            const auto* event = FindPerfEvent(state, header.sample_config.perf_event.id);
            if (!event) {
                return {};
            }
            values = {{{event->Kind, event->Unit}, ScalePerfCounterDelta(header.value)}};
            break;
        }
        case SAMPLE_TYPE_KPROBE_FINISH_TASK_SWITCH:
            values = {{{"cpu", "cycles"}, ScalePerfCounterDelta(header.value)}};
            break;
        case SAMPLE_TYPE_TRACEPOINT_SIGNAL_DELIVER:
            return {{{"signal", "count"}, 1}};
        case SAMPLE_TYPE_UPROBE:
            return {{{Output_.UprobeSampleKind.empty() ? "uprobe" : Output_.UprobeSampleKind, "count"}, 1}};
        default:
            return {};
    }
    if (Output_.IncludeWallTime) {
        values.push_back({{"wall", "seconds"}, header.timedelta});
    }
    return values;
}

void TSampleAccumulator::CheckValueTypes(TConstArrayRef<TSampleValue> values) const {
    if (ValueTypes_.empty()) {
        return;
    }
    Y_ENSURE(ValueTypes_.size() == values.size(), "sample value types do not match the selected builder");
    for (size_t i = 0; i < values.size(); ++i) {
        Y_ENSURE(ValueTypes_[i] == values[i].Type, "sample value types do not match the selected builder");
    }
}

void TSampleAccumulator::InitializeValueTypes(TConstArrayRef<TSampleValue> values) {
    if (!ValueTypes_.empty()) {
        return;
    }
    ValueTypes_.reserve(values.size());
    ValueTypeIds_.reserve(values.size());
    for (const auto& value : values) {
        const auto& [kind, unit] = value.Type;
        ValueTypes_.push_back(value.Type);
        ValueTypeIds_.push_back(Builder_.AddValueType(kind, unit));
    }
}

void TSampleAccumulator::FeedTimestamp(i64 timestampNanos) {
    if (!HasTimestampRange_) {
        HasTimestampRange_ = true;
        MinTimestampNanos_ = timestampNanos;
        MaxTimestampNanos_ = timestampNanos;
    } else {
        MinTimestampNanos_ = Min(MinTimestampNanos_, timestampNanos);
        MaxTimestampNanos_ = Max(MaxTimestampNanos_, timestampNanos);
    }
}

void TSampleAccumulator::AddCommonLabels(
    TLowLevelProfileBuilder::TSimpleSampleKeyBuilder& sampleKeyBuilder,
    const TStateSnapshot& state,
    const TProcessState* process,
    const TRawSample& sample
)
{
    const auto& header = sample.Header();
    sampleKeyBuilder.AddLabel(Builder_.AddNumericLabel(GetWellKnownLabelKey(WellKnownLabel::ProcessId), header.pid));
    sampleKeyBuilder.AddLabel(Builder_.AddNumericLabel(GetWellKnownLabelKey(WellKnownLabel::ThreadId), header.tid));
    sampleKeyBuilder.AddLabel(Builder_.AddStringLabel(GetAllWellKnownLabelKeys(WellKnownLabel::ThreadCommand)[1], Comm(header.thread_comm)));
    sampleKeyBuilder.AddLabel(Builder_.AddStringLabel(GetWellKnownLabelKey(WellKnownLabel::ProcessCommand), Comm(header.process_comm)));
    sampleKeyBuilder.AddLabel(Builder_.AddStringLabel(GetWellKnownLabelKey(WellKnownLabel::ThreadCommand), Comm(header.thread_comm)));

    const auto& cgroupNames = state.Global.Cgroups->Names;
    TStringBuf parentCgroupName;
    if (auto it = cgroupNames.find(header.parent_cgroup); it != cgroupNames.end()) {
        parentCgroupName = it->second.FullName;
    }
    sampleKeyBuilder.AddLabel(Builder_.AddStringLabel(GetWellKnownLabelKey(WellKnownLabel::Cgroup), parentCgroupName));

    if (Options_.EnableInnermostPidns) {
        sampleKeyBuilder.AddLabel(Builder_.AddNumericLabel(GetWellKnownLabelKey(WellKnownLabel::InnermostPidnsThreadId), header.innermost_pidns_tid));
        sampleKeyBuilder.AddLabel(Builder_.AddNumericLabel(GetWellKnownLabelKey(WellKnownLabel::InnermostPidnsProcessId), header.innermost_pidns_pid));
    }

    const auto cgroupIds = sample.Cgroups();
    const auto& workloads = state.Global.Cgroups->Workloads;
    const size_t cgroupCount = WorkloadCgroupCount(sample);
    for (size_t remainingCgroups = cgroupCount; remainingCgroups != 0; --remainingCgroups) {
        TStringBuf cgroupName = "<unknown cgroup>";
        if (auto it = cgroupNames.find(cgroupIds.Read(remainingCgroups - 1)); it != cgroupNames.end() && !it->second.BaseName.empty()) {
            cgroupName = it->second.BaseName;
        }
        if (remainingCgroups == cgroupCount) {
            if (auto it = workloads.find(cgroupName); it != workloads.end()) {
                sampleKeyBuilder.AddLabel(Builder_.AddStringLabel(GetWellKnownLabelKey(WellKnownLabel::Workload), it->second.ServiceName));
                sampleKeyBuilder.AddLabel(Builder_.AddStringLabel(GetWellKnownLabelKey(WellKnownLabel::Workload), it->second.ContainerName));
                continue;
            }
        }
        sampleKeyBuilder.AddLabel(Builder_.AddStringLabel(GetWellKnownLabelKey(WellKnownLabel::Workload), cgroupName));
    }
    if (process) {
        for (const auto& environment : process->Environment) {
            sampleKeyBuilder.AddLabel(Builder_.AddStringLabel(environment.Key, environment.Value));
        }
        const auto tls = sample.Tls();
        for (size_t i = 0; i < tls.Count(); ++i) {
            const auto variable = tls.Read(i);
            const auto labelKeyIt = process->TLSLabelKeys.find(variable.offset);
            if (labelKeyIt == process->TLSLabelKeys.end()) {
                continue;
            }
            TString formattedNumber;
            if (const auto value = FormatTLSValue(variable, formattedNumber)) {
                sampleKeyBuilder.AddLabel(Builder_.AddStringLabel(labelKeyIt->second, *value));
            }
        }
    }
}

} // namespace NPerforator::NAgent::NNativePipeline
