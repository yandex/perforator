#pragma once

#include "state.h"

#include <util/generic/hash.h>
#include <util/generic/strbuf.h>
#include <util/generic/string.h>
#include <util/generic/vector.h>
#include <util/system/types.h>

#include <deque>
#include <memory>
#include <shared_mutex>
#include <utility>

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

struct TEnvironmentLabel {
    TString Key;
    TString Value;
    TString MetaValue;
};

struct TProcessState {
    ui64 StartTime = 0;
    ui64 Version = 0;
    // Stable across equivalent rescans; changes when executable mappings
    // change, so runtime symbol caches do not retain a previous image.
    ui64 ImageId = 0;
    TVector<TMapping> Mappings;
    TVector<TEnvironmentLabel> Environment;
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

struct TTarget {
    ui64 Id = 0;
    ui64 Key = 0;
    bool EnableSampleTime = false;
    bool EnableInnermostPidns = false;
    TVector<TStringLabel> Labels;
    i64 CreatedTimestampNanos = 0;
    i64 StartTimestampNanos = 0;
    i64 EndTimestampNanos = 0;
    bool CollectAllPerfEvents = false;
    THashMap<ui64, TPerfEventInfo> PerfEvents;
    TUprobeBuildIdIndex Uprobes;
    bool Pending = false;
};

struct TTargetState {
    ui64 Version = 0;
    std::shared_ptr<const TTarget> WholeSystem;
    THashMap<ui32, std::shared_ptr<const TTarget>> Processes;
    THashMap<ui64, std::shared_ptr<const TTarget>> Cgroups;
    TVector<std::shared_ptr<const TTarget>> GlobalCustom;
    THashMap<ui32, TVector<std::shared_ptr<const TTarget>>> CustomByProcess;
    THashMap<ui64, TPerfEventInfo> CustomPerfEvents;
    TUprobeBuildIdIndex BaseUprobes;
    THashMap<ui64, std::shared_ptr<const TTarget>> ById;

    std::shared_ptr<const TTarget> Find(ui32 pid, ui32 tid, ui64 parentCgroup) const noexcept;
    bool Contains(ui64 targetId) const noexcept;
    bool IsPending(ui64 targetId) const noexcept;
};

struct TPerfEventState {
    ui64 Version = 0;
    THashMap<ui64, TPerfEventInfo> Events;
};

struct TSampleStateSnapshot {
    std::shared_ptr<const TProcessState> Process;
    std::shared_ptr<const TCgroupState> Cgroups;
    std::shared_ptr<const TTargetState> Targets;
    std::shared_ptr<const TPerfEventState> PerfEvents;
    std::shared_ptr<const TKernelSymbolState> KernelSymbols;
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
    // Unknown process identities match provisionally; a known mismatch has no Process.
    TSampleStateSnapshot CaptureSampleState(TPerforatorProcessKey key) const;
    std::shared_ptr<const TTargetState> GetTargets() const;

private:
    struct TProcessTombstone {
        ui64 StartTime = 0;
        ui64 Version = 0;
    };

private:
    mutable std::shared_mutex Mutex_;
    THashMap<ui32, std::shared_ptr<const TProcessState>> Processes_;
    THashMap<ui32, TProcessTombstone> ProcessTombstones_;
    std::deque<std::pair<ui32, TProcessTombstone>> ProcessTombstoneOrder_;
    std::shared_ptr<const TCgroupState> Cgroups_ = std::make_shared<TCgroupState>();
    std::shared_ptr<const TTargetState> Targets_ = std::make_shared<TTargetState>();
    std::shared_ptr<const TPerfEventState> PerfEvents_ = std::make_shared<TPerfEventState>();
    std::shared_ptr<const TKernelSymbolState> KernelSymbols_ = std::make_shared<TKernelSymbolState>();
};

TStateStore* UnwrapState(TPerforatorNativeState state);

////////////////////////////////////////////////////////////////////////////////

} // namespace NPerforator::NAgent::NNativePipeline
