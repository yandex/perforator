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
    TPerforatorLanguage Language;
    uint8_t PayloadKind;
} TPerforatorLanguageSection;

typedef struct {
    TPerforatorProcessKey Key;
    const uint64_t* NativeStack;
    size_t NativeStackSize;
    const TPerforatorLanguageSection* Languages;
    size_t LanguageCount;
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

typedef struct {
    TPerforatorRuntimeString Mapping;
    uint64_t Address;
    size_t SymbolsOffset;
    uint32_t SymbolCount;
} TPerforatorResolvedFrame;

typedef struct {
    TPerforatorLanguage Language;
    size_t FrameBegin;
    size_t FrameCount;
} TPerforatorResolvedLanguageSection;

enum TPerforatorResolveStatus {
    PerforatorResolveOk = 0,
    PerforatorResolveFailed = 1,
};

typedef struct {
    char* Arena;
    size_t ArenaBytes;
    size_t FramesOffset;
    size_t FrameCount;
    size_t NativeFrameCount;
    size_t LanguagesOffset;
    size_t LanguageCount;
} TPerforatorRuntimeResult;

// Inputs are read-only and borrowed for the call. Calls sharing a resolver must be
// serialized; its cgo handle stays live. Initialize result to zero before calling.
// On Ok, the caller owns Arena and frees it with free; all offsets are from Arena.
// Records are aligned for their C types. Native frames follow NativeStack, then
// interpreter frames follow section/frame order. Languages indexes each input
// section; native annotations have an empty interpreter range. No Go pointers are stored.
// On Failed, result is zero. Optional errorMessage receives a malloc-allocated
// error or null; the caller frees it before reuse.
int PerforatorGoResolveRuntimeFrames(
    uint64_t resolver,
    TPerforatorRuntimeRequestPtr request,
    TPerforatorRuntimeResult* result,
    char** errorMessage
);

////////////////////////////////////////////////////////////////////////////////

#ifdef __cplusplus
}
#endif
