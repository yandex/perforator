#pragma once

#include "state.h"

#include <perforator/agent/collector/progs/unwinder/contract/language_ids.h>

#include <stddef.h>
#include <stdint.h>

#ifdef __cplusplus
extern "C" {
#endif

////////////////////////////////////////////////////////////////////////////////

typedef struct {
    uint8_t Language;
    uint32_t Pid;
    uint64_t ProcessStartTime;
    uint64_t ProcessImageId;
    const void* FrameData;
    uint16_t FrameSize;
} TPerforatorInterpreterFrameRequest;

enum TPerforatorResolvedFrameFlags {
    PerforatorResolvedFrameUnsymbolized = 1u << 0,
    PerforatorResolvedFrameHasStartLine = 1u << 1,
    PerforatorResolvedFrameHasLine = 1u << 2,
    // The resolution was valid for this sample but depended on transient
    // state that should be retried on the next occurrence.
    PerforatorResolvedFrameDoNotCache = 1u << 3,
};

typedef struct {
    TPerforatorStringView Mapping;
    TPerforatorStringView Name;
    TPerforatorStringView File;
    uint64_t Address;
    int32_t StartLine;
    int32_t Line;
    uint32_t Flags;
} TPerforatorResolvedInterpreterFrame;

enum TPerforatorResolveStatus {
    PerforatorResolveOk = 0,
    PerforatorResolveInsufficientCapacity = 1,
    PerforatorResolveFailed = 2,
};

// Implemented by Go. The caller owns requests, results and arena. Calls sharing
// a resolver must be serialized; its cgo handle must remain live until they return.
// Requests and their payloads are borrowed for the call. Results are valid only
// on Ok; their strings point into arena and remain valid until it is reused.
// Buffer capacities must match available storage; null buffers have zero capacity.
// On Ok or InsufficientCapacity, requiredArenaBytes contains the required byte count.
// Initialize *resolution to zero. InsufficientCapacity retains a ready result in this
// handle; retry the same operation and requestCount with it (requests are ignored).
// Ok and Failed release and clear the handle. The caller must release it
// when stopping after InsufficientCapacity without retrying.
// Optional errorMessage receives a malloc-allocated error or null.
// The caller frees it before reuse.
int PerforatorGoResolveInterpreterFrames(
    uint64_t resolver,
    TPerforatorInterpreterFrameRequest* requests,
    size_t requestCount,
    TPerforatorResolvedInterpreterFrame* results,
    char* arena,
    size_t arenaCapacity,
    size_t* requiredArenaBytes,
    uint64_t* resolution,
    char** errorMessage
);

// Releases a live resolution handle; zero is allowed.
void PerforatorGoReleaseRuntimeResolution(uint64_t resolution);

typedef struct {
    TPerforatorProcessKey Key;
    uint8_t HasJvmMethod;
    uint64_t Address;
    uint64_t JvmMethodAddress;
} TPerforatorNativeFrameRequest;

typedef struct {
    TPerforatorStringView Name;
} TPerforatorResolvedNativeSymbol;

typedef struct {
    TPerforatorStringView JitMapping;
    uint32_t SymbolBegin;
    uint32_t SymbolCount;
    TPerforatorStringView JvmName;
} TPerforatorResolvedNativeFrame;

// The same ownership and retry contract applies. symbolCount is the required
// number of symbols on Ok or InsufficientCapacity; results index the caller-owned array.
int PerforatorGoResolveNativeFrames(
    uint64_t resolver,
    TPerforatorNativeFrameRequest* requests,
    size_t requestCount,
    TPerforatorResolvedNativeFrame* results,
    TPerforatorResolvedNativeSymbol* symbols,
    size_t symbolCapacity,
    size_t* symbolCount,
    char* arena,
    size_t arenaCapacity,
    size_t* requiredArenaBytes,
    uint64_t* resolution,
    char** errorMessage
);

////////////////////////////////////////////////////////////////////////////////

#ifdef __cplusplus
}
#endif
