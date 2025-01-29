#pragma once

#include <util/system/types.h>

#ifdef __cplusplus
extern "C" {
#endif

typedef struct {
    ui32 StartLine;
    ui32 Line;
    ui32 Column;
    ui32 Discriminator;
    char* FunctionName;
    char* DemangledFunctionName;
    char* FileName;
} TLineInfo;

void* MakeSymbolizer(char** error);

// return array of inlined function names
TLineInfo* Symbolize(
    void* symb,
    char* modulePath,
    ui64 modulePathLen,
    ui64 addr,
    ui64* linesCount,
    char** error,
    ui32 useGsym
);

void PruneCaches(void* symb);

void DestroySymbolizeResult(TLineInfo* result, ui64 linesCount);

void DestroySymbolizer(void* symb);

#ifdef __cplusplus
}
#endif

