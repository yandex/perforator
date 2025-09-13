#pragma once

#include "error.h"
#include "profile.h"

#include <stddef.h>


#ifdef __cplusplus
extern "C" {
#endif

////////////////////////////////////////////////////////////////////////////////

typedef void* TPerforatorProfileMergeManager;
typedef void* TPerforatorProfileMerger;

////////////////////////////////////////////////////////////////////////////////

TPerforatorError PerforatorMakeMergeManager(
    int threadCount,
    TPerforatorProfileMergeManager* result
);

void PerforatorDestroyMergeManager(
    TPerforatorProfileMergeManager merger
);

////////////////////////////////////////////////////////////////////////////////

TPerforatorError PerforatorMergerStart(
    TPerforatorProfileMergeManager manager,
    const char* protoOptionsBegin,
    size_t protoOptionsSize,
    TPerforatorProfileMerger* result
);

TPerforatorError PerforatorMergerAddProfile(
    TPerforatorProfileMerger merger,
    TPerforatorProfile profile
);

TPerforatorError PerforatorMergerFinish(
    TPerforatorProfileMerger merger,
    TPerforatorProfile* result
);

void PerforatorMergerDispose(
    TPerforatorProfileMerger merger
);

////////////////////////////////////////////////////////////////////////////////

#ifdef __cplusplus
}
#endif
