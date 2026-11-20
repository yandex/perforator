#include "state.hpp"

#include <perforator/lib/profile/c/error.hpp>

#include <util/generic/hash_set.h>
#include <util/generic/yexception.h>

#include <algorithm>
#include <limits>
#include <mutex>
#include <utility>

namespace NPerforator::NAgent::NNativePipeline {
namespace {

////////////////////////////////////////////////////////////////////////////////

TString CopyString(TPerforatorStringView value, TStringBuf field) {
    Y_ENSURE(value.Data || value.Size == 0, field << " has a null pointer with non-zero size");
    if (value.Size == 0) {
        return {};
    }
    return TString{value.Data, value.Size};
}

template <class T>
void ValidateArray(const T* data, size_t size, TStringBuf field) {
    Y_ENSURE(data || size == 0, field << " has a null pointer with non-zero size");
}

std::shared_ptr<const TProcessState> BuildProcessState(const TPerforatorProcessSnapshot& snapshot) {
    ValidateArray(snapshot.Mappings, snapshot.MappingCount, "process mappings");
    ValidateArray(snapshot.Environment, snapshot.EnvironmentCount, "process environment");
    ValidateArray(snapshot.TLSNames, snapshot.TLSNameCount, "process TLS names");
    Y_ENSURE(snapshot.Version != 0, "process snapshot version must be non-zero");

    auto result = std::make_shared<TProcessState>();
    result->StartTime = snapshot.Key.StartTime;
    result->Version = snapshot.Version;
    result->Mappings.reserve(snapshot.MappingCount);
    result->Environment.reserve(snapshot.EnvironmentCount);
    result->TLSLabelKeys.reserve(snapshot.TLSNameCount);

    for (size_t i = 0; i < snapshot.MappingCount; ++i) {
        const auto& mapping = snapshot.Mappings[i];
        Y_ENSURE(mapping.Begin < mapping.End, "process mapping #" << i << " has an empty or reversed range");
        result->Mappings.push_back(TMapping{
            .Begin = mapping.Begin,
            .End = mapping.End,
            .Offset = mapping.Offset,
            .FileOffset = mapping.FileOffset,
            .Path = CopyString(mapping.Path, "mapping path"),
            .BuildId = CopyString(mapping.BuildId, "mapping build id"),
        });
    }

    std::ranges::sort(result->Mappings, {}, &TMapping::Begin);
    for (size_t i = 1; i < result->Mappings.size(); ++i) {
        Y_ENSURE(result->Mappings[i - 1].End <= result->Mappings[i].Begin,
                 "process snapshot contains overlapping mappings");
    }
    for (size_t i = 0; i < snapshot.EnvironmentCount; ++i) {
        const auto& label = snapshot.Environment[i];
        result->Environment.push_back(TStringLabel{
            .Key = CopyString(label.Key, "environment label key"),
            .Value = CopyString(label.Value, "environment label value"),
        });
    }

    for (size_t i = 0; i < snapshot.TLSNameCount; ++i) {
        const auto& name = snapshot.TLSNames[i];
        auto [_, inserted] = result->TLSLabelKeys.emplace(
            name.Offset,
            "tls:" + CopyString(name.Name, "TLS name"));
        Y_ENSURE(inserted, "process snapshot contains duplicate TLS offset " << name.Offset);
    }

    return result;
}

std::shared_ptr<const TCgroupState> BuildCgroupState(const TPerforatorCgroupSnapshot& snapshot) {
    ValidateArray(snapshot.Names, snapshot.NameCount, "cgroup names");
    ValidateArray(snapshot.Workloads, snapshot.WorkloadCount, "cgroup workloads");
    Y_ENSURE(snapshot.Version != 0, "cgroup snapshot version must be non-zero");

    auto result = std::make_shared<TCgroupState>();
    result->Version = snapshot.Version;
    result->Names.reserve(snapshot.NameCount);
    result->Workloads.reserve(snapshot.WorkloadCount);

    for (size_t i = 0; i < snapshot.NameCount; ++i) {
        const auto& name = snapshot.Names[i];
        auto [_, inserted] = result->Names.emplace(
            name.Id,
            TCgroupName{
                .BaseName = CopyString(name.BaseName, "cgroup base name"),
                .FullName = CopyString(name.FullName, "cgroup full name"),
            });
        Y_ENSURE(inserted, "cgroup snapshot contains duplicate id " << name.Id);
    }

    for (size_t i = 0; i < snapshot.WorkloadCount; ++i) {
        const auto& workload = snapshot.Workloads[i];
        auto [_, inserted] = result->Workloads.emplace(
            CopyString(workload.CgroupBaseName, "workload cgroup name"),
            TWorkload{
                .ServiceName = CopyString(workload.ServiceName, "workload service name"),
                .ContainerName = CopyString(workload.ContainerName, "workload container name"),
            });
        Y_ENSURE(inserted, "cgroup snapshot contains duplicate workload key");
    }

    return result;
}

std::shared_ptr<const TProfilingTargetState> BuildProfilingTargetState(const TPerforatorTargetSnapshot& snapshot) {
    ValidateArray(snapshot.BaseUprobes, snapshot.BaseUprobeCount, "base target uprobes");
    ValidateArray(snapshot.Targets, snapshot.TargetCount, "profiling targets");
    Y_ENSURE(snapshot.Version != 0, "target snapshot version must be non-zero");

    auto result = std::make_shared<TProfilingTargetState>();
    result->Version = snapshot.Version;
    result->ById.reserve(snapshot.TargetCount);

    for (size_t i = 0; i < snapshot.BaseUprobeCount; ++i) {
        const auto& uprobe = snapshot.BaseUprobes[i];
        result->BaseUprobes[CopyString(uprobe.BuildId, "base uprobe build id")][uprobe.Offset].push_back({
            .ProfileName = CopyString(uprobe.ProfileName, "base uprobe profile name"),
            .SampleKind = CopyString(uprobe.SampleKind, "base uprobe sample kind"),
        });
    }

    for (size_t i = 0; i < snapshot.TargetCount; ++i) {
        const auto& source = snapshot.Targets[i];
        ValidateArray(source.Labels, source.LabelCount, "target labels");
        ValidateArray(source.PerfEvents, source.PerfEventCount, "target perf events");
        ValidateArray(source.Uprobes, source.UprobeCount, "target uprobes");
        Y_ENSURE(source.Id != 0, "target id must be non-zero");

        auto target = std::make_shared<TProfilingTarget>();
        Y_ENSURE(result->ById.emplace(source.Id, target).second, "target snapshot contains duplicate id " << source.Id);
        target->Id = source.Id;
        target->Pending = source.Pending != 0;
        target->CreatedTimestampNanos = source.CreatedTimestampNanos;
        target->StartTimestampNanos = source.StartTimestampNanos;
        target->EndTimestampNanos = source.EndTimestampNanos;
        target->CollectAllPerfEvents = source.CollectAllPerfEvents != 0;
        auto& options = target->ProfileOptions;
        options.EnableAbsoluteTimestampLabel = source.EnableSampleTime != 0;
        options.EnableInnermostPidns = source.EnableInnermostPidns != 0;
        options.Labels.reserve(source.LabelCount);
        THashSet<TString> labelKeys;
        for (size_t j = 0; j < source.LabelCount; ++j) {
            TString key = CopyString(source.Labels[j].Key, "target label key");
            Y_ENSURE(labelKeys.insert(key).second, "target contains duplicate label " << key);
            options.Labels.push_back({
                .Key = std::move(key),
                .Value = CopyString(source.Labels[j].Value, "target label value"),
            });
        }
        for (size_t j = 0; j < source.PerfEventCount; ++j) {
            const auto& event = source.PerfEvents[j];
            auto [_, inserted] = target->PerfEvents.emplace(
                event.Id,
                TPerfEventInfo{
                    .Kind = CopyString(event.Kind, "target perf event kind"),
                    .Unit = CopyString(event.Unit, "target perf event unit"),
                });
            Y_ENSURE(inserted, "target contains duplicate perf event id " << event.Id);
        }
        for (size_t j = 0; j < source.UprobeCount; ++j) {
            const auto& uprobe = source.Uprobes[j];
            target->Uprobes[CopyString(uprobe.BuildId, "target uprobe build id")][uprobe.Offset].push_back({
                .ProfileName = CopyString(uprobe.ProfileName, "target uprobe profile name"),
                .SampleKind = CopyString(uprobe.SampleKind, "target uprobe sample kind"),
            });
        }

        switch (source.Kind) {
            case PERFORATOR_TARGET_WHOLE_SYSTEM:
                Y_ENSURE(source.Key == 0, "whole-system target key must be zero");
                Y_ENSURE(!result->WholeSystem, "target snapshot contains multiple whole-system targets");
                result->WholeSystem = target.get();
                break;
            case PERFORATOR_TARGET_PROCESS: {
                Y_ENSURE(source.Key <= std::numeric_limits<ui32>::max(), "process target PID does not fit uint32");
                auto [_, inserted] = result->Processes.emplace(static_cast<ui32>(source.Key), target.get());
                Y_ENSURE(inserted, "target snapshot contains duplicate process target");
                break;
            }
            case PERFORATOR_TARGET_CGROUP: {
                auto [_, inserted] = result->Cgroups.emplace(source.Key, target.get());
                Y_ENSURE(inserted, "target snapshot contains duplicate cgroup target");
                break;
            }
            case PERFORATOR_TARGET_CUSTOM:
                Y_ENSURE(source.Key <= std::numeric_limits<ui32>::max(),
                         "custom target PID does not fit uint32");
                Y_ENSURE(target->StartTimestampNanos == 0 || target->EndTimestampNanos == 0 ||
                             target->StartTimestampNanos <= target->EndTimestampNanos,
                         "custom target has a reversed time interval");
                for (const auto& [id, event] : target->PerfEvents) {
                    auto [existing, inserted] = result->CustomPerfEvents.emplace(id, event);
                    Y_ENSURE(inserted ||
                                 (existing->second.Kind == event.Kind && existing->second.Unit == event.Unit),
                             "custom targets disagree on perf event " << id);
                }
                if (source.Key == 0) {
                    result->GlobalCustom.push_back(target.get());
                } else {
                    result->CustomByProcess[static_cast<ui32>(source.Key)].push_back(target.get());
                }
                break;
            default:
                ythrow yexception() << "unknown target kind " << static_cast<unsigned>(source.Kind);
        }
    }

    return result;
}

std::shared_ptr<const TPerfEventState> BuildPerfEventState(const TPerforatorPerfEventSnapshot& snapshot) {
    ValidateArray(snapshot.Events, snapshot.EventCount, "perf events");
    Y_ENSURE(snapshot.Version != 0, "perf event snapshot version must be non-zero");

    auto result = std::make_shared<TPerfEventState>();
    result->Version = snapshot.Version;
    result->Events.reserve(snapshot.EventCount);
    for (size_t i = 0; i < snapshot.EventCount; ++i) {
        const auto& event = snapshot.Events[i];
        auto [_, inserted] = result->Events.emplace(
            event.Id,
            TPerfEventInfo{
                .Kind = CopyString(event.Kind, "perf event kind"),
                .Unit = CopyString(event.Unit, "perf event unit"),
            });
        Y_ENSURE(inserted, "perf event snapshot contains duplicate id " << event.Id);
    }
    return result;
}

////////////////////////////////////////////////////////////////////////////////

} // anonymous namespace

const TMapping* TProcessState::ResolveMapping(ui64 address) const noexcept {
    auto it = std::upper_bound(
        Mappings.begin(), Mappings.end(), address,
        [](ui64 value, const TMapping& mapping) {
            return value < mapping.Begin;
        });
    if (it == Mappings.begin()) {
        return nullptr;
    }
    --it;
    return address < it->End ? &*it : nullptr;
}

TStringBuf TKernelSymbolState::Resolve(ui64 address) const noexcept {
    auto it = std::upper_bound(
        Symbols.begin(), Symbols.end(), address,
        [](ui64 value, const TKernelSymbol& symbol) {
            return value < symbol.Address;
        });
    if (it == Symbols.begin()) {
        return "unknown";
    }
    return (--it)->Name;
}

const TVector<TUprobeOutput>* FindUprobes(
    const TUprobeBuildIdIndex& uprobes,
    TStringBuf buildId,
    ui64 offset
)
{
    auto binary = uprobes.find(buildId);
    if (binary == uprobes.end()) {
        return nullptr;
    }
    auto outputs = binary->second.find(offset);
    return outputs == binary->second.end() ? nullptr : &outputs->second;
}

const TProfilingTarget* TProfilingTargetState::Find(ui32 pid, ui32 tid, ui64 parentCgroup) const noexcept {
    if (WholeSystem) {
        return WholeSystem;
    }
    if (auto it = Processes.find(pid); it != Processes.end()) {
        return it->second;
    }
    if (auto it = Processes.find(tid); it != Processes.end()) {
        return it->second;
    }
    if (auto it = Cgroups.find(parentCgroup); it != Cgroups.end()) {
        return it->second;
    }
    return nullptr;
}

std::shared_ptr<const TProfilingTarget> TProfilingTargetState::Get(ui64 targetId) const noexcept {
    auto it = ById.find(targetId);
    return it == ById.end() ? nullptr : it->second;
}

bool TProfilingTargetState::Contains(ui64 targetId) const noexcept {
    return ById.contains(targetId);
}

bool TProfilingTargetState::IsPending(ui64 targetId) const noexcept {
    auto it = ById.find(targetId);
    return it != ById.end() && it->second->Pending;
}

void TStateStore::ReplaceProcess(const TPerforatorProcessSnapshot& snapshot) {
    auto process = BuildProcessState(snapshot);

    std::unique_lock guard{Mutex_};
    auto it = Processes_->find(snapshot.Key.Pid);
    if (it != Processes_->end()) {
        const auto& current = *it->second;
        if (current.StartTime > snapshot.Key.StartTime ||
            (current.StartTime == snapshot.Key.StartTime && current.Version >= snapshot.Version))
        {
            return;
        }
    }
    if (!Processes_.unique()) {
        Processes_ = std::make_shared<TProcessStates>(*Processes_);
    }
    (*Processes_)[snapshot.Key.Pid] = std::move(process);
}

void TStateStore::RemoveProcess(TPerforatorProcessKey key, ui64 version) {
    auto [pid, startTime] = key;
    Y_ENSURE(version != 0, "process removal version must be non-zero");
    std::unique_lock guard{Mutex_};
    auto it = Processes_->find(pid);
    if (it != Processes_->end()) {
        const auto& current = *it->second;
        if ((startTime != 0 && current.StartTime != 0 && current.StartTime > startTime) ||
            ((startTime == 0 || current.StartTime == 0 || current.StartTime == startTime) && current.Version >= version))
        {
            return;
        }
        if (!Processes_.unique()) {
            Processes_ = std::make_shared<TProcessStates>(*Processes_);
        }
        Processes_->erase(pid);
    }
}

void TStateStore::ReplaceKernelSymbols(const TPerforatorKernelSymbolSnapshot& snapshot) {
    ValidateArray(snapshot.Symbols, snapshot.SymbolCount, "kernel symbols");
    auto result = std::make_shared<TKernelSymbolState>();
    result->Symbols.reserve(snapshot.SymbolCount);
    for (size_t i = 0; i < snapshot.SymbolCount; ++i) {
        Y_ENSURE(i == 0 || snapshot.Symbols[i - 1].Address <= snapshot.Symbols[i].Address,
                 "kernel symbol snapshot is not address-sorted");
        result->Symbols.push_back({
            .Address = snapshot.Symbols[i].Address,
            .Name = CopyString(snapshot.Symbols[i].Name, "kernel symbol name"),
        });
    }
    std::unique_lock guard{Mutex_};
    Global_.KernelSymbols = std::move(result);
}

void TStateStore::ReplaceCgroups(const TPerforatorCgroupSnapshot& snapshot) {
    auto cgroups = BuildCgroupState(snapshot);

    std::unique_lock guard{Mutex_};
    if (Global_.Cgroups->Version >= snapshot.Version) {
        return;
    }
    Global_.Cgroups = std::move(cgroups);
}

void TStateStore::ReplaceTargets(const TPerforatorTargetSnapshot& snapshot) {
    auto targets = BuildProfilingTargetState(snapshot);

    std::unique_lock guard{Mutex_};
    if (Global_.Targets->Version >= snapshot.Version) {
        return;
    }
    Global_.Targets = std::move(targets);
}

void TStateStore::ReplacePerfEvents(const TPerforatorPerfEventSnapshot& snapshot) {
    auto events = BuildPerfEventState(snapshot);
    std::unique_lock guard{Mutex_};
    if (Global_.PerfEvents->Version >= snapshot.Version) {
        return;
    }
    Global_.PerfEvents = std::move(events);
}

std::shared_ptr<const TProcessState> TStateSnapshot::FindProcess(TPerforatorProcessKey key) const {
    const auto [pid, startTime] = key;
    if (Processes) {
        if (auto it = Processes->find(pid);
            it != Processes->end() &&
            (it->second->StartTime == 0 || startTime == 0 || it->second->StartTime == startTime))
        {
            return it->second;
        }
    }
    return {};
}

TStateSnapshot TStateStore::Snapshot() const {
    std::shared_lock guard{Mutex_};
    return {
        .Processes = Processes_,
        .Global = Global_,
    };
}

std::shared_ptr<const TProfilingTargetState> TStateStore::GetTargets() const {
    std::shared_lock guard{Mutex_};
    return Global_.Targets;
}

TStateStore* UnwrapState(TPerforatorNativeState state) {
    return reinterpret_cast<TStateStore*>(state);
}

////////////////////////////////////////////////////////////////////////////////

using namespace NPerforator::NProfile::NCWrapper;

extern "C" {

TPerforatorError PerforatorNativeStateCreate(TPerforatorNativeState* result) {
    return InterceptExceptions([&] {
        Y_ENSURE(result, "expected a non-null native state output pointer");
        *result = new TStateStore{};
    });
}

void PerforatorNativeStateDispose(TPerforatorNativeState state) {
    delete UnwrapState(state);
}

TPerforatorError PerforatorNativeStateReplaceProcess(
    TPerforatorNativeState state,
    const TPerforatorProcessSnapshot* snapshot
)
{
    return InterceptExceptions([&] {
        Y_ENSURE(state, "expected a non-null native state");
        Y_ENSURE(snapshot, "expected a non-null process snapshot");
        UnwrapState(state)->ReplaceProcess(*snapshot);
    });
}

TPerforatorError PerforatorNativeStateRemoveProcess(
    TPerforatorNativeState state,
    TPerforatorProcessKey key,
    uint64_t version
)
{
    return InterceptExceptions([&] {
        Y_ENSURE(state, "expected a non-null native state");
        UnwrapState(state)->RemoveProcess(key, version);
    });
}

TPerforatorError PerforatorNativeStateReplaceKernelSymbols(
    TPerforatorNativeState state,
    const TPerforatorKernelSymbolSnapshot* snapshot
)
{
    return InterceptExceptions([&] {
        Y_ENSURE(state, "expected a non-null native state");
        Y_ENSURE(snapshot, "expected a non-null kernel symbol snapshot");
        UnwrapState(state)->ReplaceKernelSymbols(*snapshot);
    });
}

TPerforatorError PerforatorNativeStateReplaceCgroups(
    TPerforatorNativeState state,
    const TPerforatorCgroupSnapshot* snapshot
)
{
    return InterceptExceptions([&] {
        Y_ENSURE(state, "expected a non-null native state");
        Y_ENSURE(snapshot, "expected a non-null cgroup snapshot");
        UnwrapState(state)->ReplaceCgroups(*snapshot);
    });
}

TPerforatorError PerforatorNativeStateReplaceTargets(
    TPerforatorNativeState state,
    const TPerforatorTargetSnapshot* snapshot
)
{
    return InterceptExceptions([&] {
        Y_ENSURE(state, "expected a non-null native state");
        Y_ENSURE(snapshot, "expected a non-null target snapshot");
        UnwrapState(state)->ReplaceTargets(*snapshot);
    });
}

TPerforatorError PerforatorNativeStateReplacePerfEvents(
    TPerforatorNativeState state,
    const TPerforatorPerfEventSnapshot* snapshot
)
{
    return InterceptExceptions([&] {
        Y_ENSURE(state, "expected a non-null native state");
        Y_ENSURE(snapshot, "expected a non-null perf event snapshot");
        UnwrapState(state)->ReplacePerfEvents(*snapshot);
    });
}

} // extern "C"

} // namespace NPerforator::NAgent::NNativePipeline
