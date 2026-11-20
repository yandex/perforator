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
    const void* Data;
    size_t Size;
    uint16_t FrameSize;
    EPerforatorLanguage Language;
} TPerforatorLanguageSection;

typedef struct {
    TPerforatorProcessKey Key;
    const uint64_t* NativeStack;
    size_t NativeStackSize;
    const TPerforatorLanguageSection* Languages;
    size_t LanguagesSize;
} TPerforatorRuntimeRequest;

typedef const TPerforatorRuntimeRequest* TPerforatorRuntimeRequestPtr;

typedef struct {
    size_t Offset;
    size_t Size;
} TPerforatorRuntimeString;

typedef struct {
    TPerforatorRuntimeString Name;
    TPerforatorRuntimeString File;
    uint32_t StartLine;
    uint32_t Line;
} TPerforatorResolvedSymbol;

// Native addresses require DSO lookup. Runtime locations are used as returned,
// including results with an empty inline chain.
enum EPerforatorFrameRepresentation {
    PerforatorFrameNativeAddress = 0,
    PerforatorFrameRuntimeLocation = 1,
};

typedef struct {
    TPerforatorRuntimeString Mapping;
    uint64_t Address;
    size_t SymbolsOffset;
    uint32_t SymbolCount;
    uint8_t Representation; // EPerforatorFrameRepresentation.
    uint8_t Language;       // EPerforatorLanguage, or PERFORATOR_LANGUAGE_COUNT for native/JIT.
} TPerforatorResolvedFrame;

enum EPerforatorResolveStatus {
    PerforatorResolveOk = 0,
    PerforatorResolveFailed = 1,
};

typedef struct {
    char* Arena;
    size_t ArenaBytes;
    size_t FramesOffset;
    size_t FrameCount;
    size_t UserFrameCount; // One native/JIT/JVM location per input native slot, including empty slots.
} TPerforatorRuntimeResult;

// Inputs are read-only and borrowed for the call. Calls sharing a resolver must be
// serialized; its cgo handle stays live. Initialize result to zero before calling.
// On Ok, the caller owns Arena and frees it with free; offsets are from Arena.
// Records are aligned for their C types. User frames precede unbound language
// frames in input section order; both ranges are leaf-first. Each frame carries
// its source language for validation. NativeAddress requires C++ DSO lookup;
// RuntimeLocation is already resolved. No Go pointers are stored.
// On Failed, result is zero. Optional errorMessage receives a malloc-allocated
// error or null; the caller frees it before reuse.
int PerforatorGoResolveRuntimeFrames(
    uint64_t handle,
    TPerforatorRuntimeRequestPtr request,
    TPerforatorRuntimeResult* result,
    char** errorMessage
);

////////////////////////////////////////////////////////////////////////////////

#ifdef __cplusplus
}
#endif
