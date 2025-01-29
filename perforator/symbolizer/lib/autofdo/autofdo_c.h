#pragma once

#include <util/system/types.h>

#ifdef __cplusplus
extern "C" {
#endif

void* MakeBatchBuilder(ui64 buildersCount, const char* buildId);

void DestroyBatchBuilder(void* builder);

void AddProfile(void* builder, ui64 builderIndex, const char* profileBytes, ui64 profileBytesLen);

void Finalize(
    void* builder,
    ui64* totalProfiles,
    ui64* totalBranches, ui64* totalSamples, ui64* bogusLbrEntries,
    ui64* branchCountMapSize, ui64* rangeCountMapSize, ui64* addressCountMapSize,
    char** autofdoInput, char** boltInput);

ui64 GetBinaryExecutableBytes(const char* binaryPath);

///////////////////////////////////////////////////////////////////////////////////////////

void* MakeBatchBuildIdGuesser(ui64 guessersCount);

void DestroyBatchBuildIdGuesser(void* guesser);

void FeedProfileIntoGuesser(void* guesser, ui64 guesserIndex, const char* profileBytes, ui64 profileBytesLen);

const char* TryGuessBuildID(void* guesser);

#ifdef __cplusplus
}
#endif

