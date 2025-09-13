#pragma once

#include <util/system/types.h>

#ifdef __cplusplus
extern "C" {
#endif

void *MakeServicePerfTopAggregator();

void DestroyServicePerfTopAggregator(void *aggregator);

void InitializeSymbolizerForServicePerfTopAggregator(
    void *aggregator,
    const char* buildIdBytes, ui64 buildIdBytesLen,
    const char* gsymPathBytes, ui64 gsymPathBytesLen
);

void AddProfileIntoServicePerfTopAggregator(
    void *aggregator,
    const char* service,
    ui64 serviceLen,
    const char* profileBytes,
    ui64 profileBytesLen
);

void MergeServicePerfTopAggregators(void *aggregator, void *otherAggregator);

void FinalizeServicePerfTopAggregator(
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
