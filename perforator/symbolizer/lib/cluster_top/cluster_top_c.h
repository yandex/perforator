#pragma once

#include <util/system/types.h>

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

void AddProfileIntoPerfTopAggregator(
    void *aggregator,
    const char* profileBytes,
    ui64 profileBytesLen
);

void MergePerfTopAggregators(void *aggregator, void *otherAggregator);

void FinalizePerfTopAggregator(
    void *aggregator,
    ui64* nEntries,
    const char*** functions,
    // selfCycles would be an array of 16 * nEntries bytes,
    // where each block of 16 bytes is a big-endian representation of a ui128
    char** selfCycles,
    // ditto
    char** cumulativeCycles
);

#ifdef __cplusplus
}
#endif
