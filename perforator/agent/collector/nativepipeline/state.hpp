#pragma once

#include "state.h"

#include <util/generic/hash.h>
#include <util/generic/strbuf.h>
#include <util/generic/string.h>
#include <util/generic/vector.h>
#include <util/system/types.h>

#include <memory>
#include <shared_mutex>

namespace NPerforator::NAgent::NNativePipeline {

////////////////////////////////////////////////////////////////////////////////

struct TMapping {
    ui64 Begin = 0;
    ui64 End = 0;
    ui64 Offset = 0;
    ui64 FileOffset = 0;
    TString Path;
    TString BuildId;
};

struct TStringLabel {
    TString Key;
    TString Value;
};

struct TProcessState {
    ui64 StartTime = 0;
    ui64 Version = 0;
    TVector<TMapping> Mappings;
    TVector<TStringLabel> Environment;
    THashMap<ui64, TString> TLSLabelKeys;

    const TMapping* ResolveMapping(ui64 address) const noexcept;
};

struct TCgroupName {
    TString BaseName;
    TString FullName;
};

struct TKernelSymbol {
    ui64 Address = 0;
    TString Name;
};

struct TKernelSymbolState {
    TVector<TKernelSymbol> Symbols;

    TStringBuf Resolve(ui64 address) const noexcept;
};

struct TWorkload {
    TString ServiceName;
    TString ContainerName;
};

struct TCgroupState {
    ui64 Version = 0;
    THashMap<ui64, TCgroupName> Names;
    THashMap<TString, TWorkload> Workloads;
};

struct TPerfEventInfo {
    TString Kind;
    TString Unit;
};

struct TUprobeOutput {
    TString ProfileName;
    TString SampleKind;
};

using TUprobeOffsetIndex = THashMap<ui64, TVector<TUprobeOutput>>;
using TUprobeBuildIdIndex = THashMap<TString, TUprobeOffsetIndex>;

const TVector<TUprobeOutput>* FindUprobes(
    const TUprobeBuildIdIndex& uprobes,
    TStringBuf buildId,
    ui64 offset
);

struct TProfileOptions {
    bool EnableAbsoluteTimestampLabel = false;
    bool EnableInnermostPidns = false;
    TVector<TStringLabel> Labels;
};

struct TProfilingTarget {
    ui64 Id = 0;
    TProfileOptions ProfileOptions;
    i64 CreatedTimestampNanos = 0;
    i64 StartTimestampNanos = 0;
    i64 EndTimestampNanos = 0;
    bool CollectAllPerfEvents = false;
    THashMap<ui64, TPerfEventInfo> PerfEvents;
    TUprobeBuildIdIndex Uprobes;
    bool Pending = false;
};

struct TProfilingTargetState {
    ui64 Version = 0;
    // Secondary indexes borrow targets owned by ById.
    const TProfilingTarget* WholeSystem = nullptr;
    THashMap<ui32, const TProfilingTarget*> Processes;
    THashMap<ui64, const TProfilingTarget*> Cgroups;
    TVector<const TProfilingTarget*> GlobalCustom;
    THashMap<ui32, TVector<const TProfilingTarget*>> CustomByProcess;
    THashMap<ui64, TPerfEventInfo> CustomPerfEvents;
    TUprobeBuildIdIndex BaseUprobes;
    THashMap<ui64, std::shared_ptr<const TProfilingTarget>> ById;

    // Find borrows for this snapshot's lifetime; Get retains a selected target.
    const TProfilingTarget* Find(ui32 pid, ui32 tid, ui64 parentCgroup) const noexcept;
    std::shared_ptr<const TProfilingTarget> Get(ui64 targetId) const noexcept;
    bool Contains(ui64 targetId) const noexcept;
    bool IsPending(ui64 targetId) const noexcept;
};

struct TPerfEventState {
    ui64 Version = 0;
    THashMap<ui64, TPerfEventInfo> Events;
};

using TProcessStates = THashMap<ui32, std::shared_ptr<const TProcessState>>;

struct TGlobalProfilingState {
    std::shared_ptr<const TCgroupState> Cgroups;
    std::shared_ptr<const TProfilingTargetState> Targets;
    std::shared_ptr<const TPerfEventState> PerfEvents;
    std::shared_ptr<const TKernelSymbolState> KernelSymbols;
};

struct TStateSnapshot {
    std::shared_ptr<const TProcessStates> Processes;
    TGlobalProfilingState Global;

    // Unknown identities match provisionally; a known mismatch has no process.
    std::shared_ptr<const TProcessState> FindProcess(TPerforatorProcessKey key) const;
};

class TStateStore {
public:
    void ReplaceProcess(const TPerforatorProcessSnapshot& snapshot);
    void RemoveProcess(TPerforatorProcessKey key, ui64 version);
    void ReplaceKernelSymbols(const TPerforatorKernelSymbolSnapshot& snapshot);
    void ReplaceCgroups(const TPerforatorCgroupSnapshot& snapshot);
    void ReplaceTargets(const TPerforatorTargetSnapshot& snapshot);
    void ReplacePerfEvents(const TPerforatorPerfEventSnapshot& snapshot);

    // Captured immutable objects may outlive subsequent publications and the store.
    TStateSnapshot Snapshot() const;
    std::shared_ptr<const TProfilingTargetState> GetTargets() const;

private:
    mutable std::shared_mutex Mutex_;
    std::shared_ptr<TProcessStates> Processes_ = std::make_shared<TProcessStates>();
    TGlobalProfilingState Global_{
        .Cgroups = std::make_shared<TCgroupState>(),
        .Targets = std::make_shared<TProfilingTargetState>(),
        .PerfEvents = std::make_shared<TPerfEventState>(),
        .KernelSymbols = std::make_shared<TKernelSymbolState>(),
    };
};

TStateStore* UnwrapState(TPerforatorNativeState state);

////////////////////////////////////////////////////////////////////////////////

} // namespace NPerforator::NAgent::NNativePipeline
