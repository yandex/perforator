#pragma once

#include <util/system/types.h>
#include <perforator/lib/profile/c/error.h>

#ifdef __cplusplus
extern "C" {
#endif

void *MakePerfTopAggregator();

void DestroyPerfTopAggregator(void *aggregator);

void InitializeSymbolizerForPerfTopAggregator(
    void *aggregator,
    const char* buildIdBytes, ui64 buildIdBytesLen,
    const char* gsymPathBytes, ui64 gsymPathBytesLen
);

TPerforatorError AddYaprofIntoPerfTopAggregator(
    void *aggregator,
    const char* profileBytes,
    ui64 profileBytesLen
);

void MergePerfTopAggregators(void *aggregator, void *otherAggregator);

typedef struct {
    char* Name;
    char* BuildId;
    char* SourceFile;
    ui32 SourceLine;
    ui32 SourceColumn;
    // Big-endian UInt128 counters.
    char SelfCycles[16];
    char CumulativeCycles[16];
} TClusterTopFunction;

void FinalizePerfTopAggregator(void* aggregator, ui64* nEntries, TClusterTopFunction** functions);

void FreePerfTopAggregatorResult(ui64 nEntries, TClusterTopFunction* functions);

#ifdef __cplusplus
}
#endif
