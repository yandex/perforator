#pragma once

#include <perforator/lib/profile/c/error.h>

#include <stddef.h>
#include <stdint.h>

#ifdef __cplusplus
extern "C" {
#endif

////////////////////////////////////////////////////////////////////////////////

typedef void* TPerforatorNativeState;

typedef struct {
    const char* Data;
    size_t Size;
} TPerforatorStringView;

// Executable mapping of a process.
typedef struct {
    uint64_t Begin;
    uint64_t End;
    uint64_t Offset;
    uint64_t FileOffset;
    TPerforatorStringView Path;
    TPerforatorStringView BuildId;
} TPerforatorProcessMapping;

typedef struct {
    TPerforatorStringView Key;
    TPerforatorStringView Value;
} TPerforatorStringLabel;

typedef struct {
    uint64_t Offset;
    TPerforatorStringView Name;
} TPerforatorTLSName;

// Identifies one process lifetime. StartTime is zero until known.
typedef struct {
    uint32_t Pid;
    uint64_t StartTime;
} TPerforatorProcessKey;

// Process state is replaced as one immutable unit. Version must grow for every
// discovery/rescan of the same process identity. Stale updates are ignored;
// replacing an unknown-identity tombstone requires a known StartTime and a newer Version.
typedef struct {
    TPerforatorProcessKey Key;
    uint64_t Version;

    const TPerforatorProcessMapping* Mappings;
    size_t MappingCount;

    const TPerforatorStringLabel* Environment;
    size_t EnvironmentCount;

    // Names from the main binary, keyed by the TLS offsets reported in samples.
    const TPerforatorTLSName* TLSNames;
    size_t TLSNameCount;
} TPerforatorProcessSnapshot;

typedef struct {
    uint64_t Address;
    TPerforatorStringView Name;
} TPerforatorKernelSymbol;

typedef struct {
    const TPerforatorKernelSymbol* Symbols;
    size_t SymbolCount;
} TPerforatorKernelSymbolSnapshot;

typedef struct {
    uint64_t Id;
    TPerforatorStringView BaseName;
    TPerforatorStringView FullName;
} TPerforatorCgroupName;

typedef struct {
    TPerforatorStringView CgroupBaseName;
    TPerforatorStringView ServiceName;
    TPerforatorStringView ContainerName;
} TPerforatorWorkload;

typedef struct {
    uint64_t Version;
    const TPerforatorCgroupName* Names;
    size_t NameCount;
    const TPerforatorWorkload* Workloads;
    size_t WorkloadCount;
} TPerforatorCgroupSnapshot;

typedef uint8_t TPerforatorTargetKind;
enum {
    PERFORATOR_TARGET_WHOLE_SYSTEM = 1,
    PERFORATOR_TARGET_PROCESS = 2,
    PERFORATOR_TARGET_CGROUP = 3,
    PERFORATOR_TARGET_CUSTOM = 4,
};

typedef struct {
    uint64_t Id;
    TPerforatorStringView Kind;
    TPerforatorStringView Unit;
} TPerforatorPerfEvent;

typedef struct {
    TPerforatorStringView BuildId;
    uint64_t Offset;
    TPerforatorStringView ProfileName;
    TPerforatorStringView SampleKind;
} TPerforatorTargetUprobe;

typedef struct {
    uint64_t Id;
    TPerforatorTargetKind Kind;
    // PID for PROCESS, cgroup id for CGROUP, zero for WHOLE_SYSTEM.
    uint64_t Key;
    uint8_t EnableSampleTime;
    uint8_t EnableInnermostPidns;
    const TPerforatorStringLabel* Labels;
    size_t LabelCount;
    int64_t CreatedTimestampNanos;
    int64_t StartTimestampNanos;
    int64_t EndTimestampNanos;
    uint8_t CollectAllPerfEvents;
    const TPerforatorPerfEvent* PerfEvents;
    size_t PerfEventCount;
    const TPerforatorTargetUprobe* Uprobes;
    size_t UprobeCount;
    // While pending, collect samples but do not publish profiles.
    // Clear after the target's event sources have been successfully activated.
    uint8_t Pending;
} TPerforatorTarget;

typedef struct {
    uint64_t Version;
    const TPerforatorTargetUprobe* BaseUprobes;
    size_t BaseUprobeCount;
    const TPerforatorTarget* Targets;
    size_t TargetCount;
} TPerforatorTargetSnapshot;

typedef struct {
    uint64_t Version;
    const TPerforatorPerfEvent* Events;
    size_t EventCount;
} TPerforatorPerfEventSnapshot;

////////////////////////////////////////////////////////////////////////////////

// TPerforatorNativeState is a handle to TStateStore, the C++ metadata store.
// Go creates and disposes it; disposal requires all store users to stop.
//
// Go owns input snapshots and their buffers. Replace* copies the data into C++
// and replaces one process or a whole metadata category under synchronization.
// Inputs are needed only until the call returns. Invalid or stale updates leave
// published data unchanged.
//
// TSampleStateSnapshot holds shared references to immutable C++ snapshots.
// Readers keep old snapshots alive across updates, even after store disposal.
//
// ProcessTombstones_ records removed processes to reject late snapshots.
// The store owns these records; ProcessTombstoneOrder_ limits their retention.
TPerforatorError PerforatorNativeStateCreate(TPerforatorNativeState* result);

void PerforatorNativeStateDispose(TPerforatorNativeState state);

TPerforatorError PerforatorNativeStateReplaceProcess(
    TPerforatorNativeState state,
    const TPerforatorProcessSnapshot* snapshot
);

// Removes this lifetime or an older one. For the same or unknown identity,
// Version must exceed the current publication or removal version. Older deaths
// are ignored. An unknown start time preserves the last known process identity.
// Tombstones reject late publications of retired lifetimes while retained;
// retention is bounded to the latest 65536 accepted removal records.
TPerforatorError PerforatorNativeStateRemoveProcess(
    TPerforatorNativeState state,
    TPerforatorProcessKey key,
    uint64_t version
);

TPerforatorError PerforatorNativeStateReplaceKernelSymbols(
    TPerforatorNativeState state,
    const TPerforatorKernelSymbolSnapshot* snapshot
);

TPerforatorError PerforatorNativeStateReplaceCgroups(
    TPerforatorNativeState state,
    const TPerforatorCgroupSnapshot* snapshot
);

TPerforatorError PerforatorNativeStateReplaceTargets(
    TPerforatorNativeState state,
    const TPerforatorTargetSnapshot* snapshot
);

TPerforatorError PerforatorNativeStateReplacePerfEvents(
    TPerforatorNativeState state,
    const TPerforatorPerfEventSnapshot* snapshot
);

////////////////////////////////////////////////////////////////////////////////

#ifdef __cplusplus
}
#endif
