#pragma once

#include <stddef.h>


#ifdef __cplusplus
extern "C" {
#endif

////////////////////////////////////////////////////////////////////////////////

typedef void* TPerforatorError;

const char* PerforatorErrorString(TPerforatorError err);

void PerforatorErrorDispose(TPerforatorError err);

////////////////////////////////////////////////////////////////////////////////

#ifdef __cplusplus
}
#endif
