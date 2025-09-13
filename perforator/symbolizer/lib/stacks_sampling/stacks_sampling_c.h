#pragma once

#include <util/system/types.h>

#ifdef __cplusplus
extern "C" {
#endif

void *CreateAggregatingStacksSampler(ui64 rate);

void DestroyAggregatingStacksSampler(void* sampler);

void AddProfileIntoAggregatingStacksSampler(
    void* sampler,
    const char* profileBytes,
    ui64 profileBytesLen
);

void ExtractResultingProfileFromSampler(
    void* sampler,
    const char** dataPtr,
    ui64* dataLenPtr,
    ui64* isEmpty
);

#ifdef __cplusplus
}
#endif
