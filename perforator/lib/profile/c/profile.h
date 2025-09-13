#pragma once

#include "error.h"
#include "string.h"

#include <stddef.h>


#ifdef __cplusplus
extern "C" {
#endif

////////////////////////////////////////////////////////////////////////////////

typedef void* TPerforatorProfile;

////////////////////////////////////////////////////////////////////////////////

void PerforatorProfileDispose(TPerforatorProfile profile);

TPerforatorError PerforatorProfileParse(const char* ptr, size_t size, TPerforatorProfile* result);

TPerforatorError PerforatorProfileParsePProf(const char* ptr, size_t size, TPerforatorProfile* result);

TPerforatorError PerforatorProfileSerialize(TPerforatorProfile profile, TPerforatorString* result);

TPerforatorError PerforatorProfileSerializePProf(TPerforatorProfile profile, TPerforatorString* result);

////////////////////////////////////////////////////////////////////////////////

#ifdef __cplusplus
}
#endif
