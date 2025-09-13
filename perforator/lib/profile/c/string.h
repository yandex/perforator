#pragma once

#include <stddef.h>


#ifdef __cplusplus
extern "C" {
#endif

////////////////////////////////////////////////////////////////////////////////

typedef void* TPerforatorString;

const char* PerforatorStringData(TPerforatorString str);

size_t PerforatorStringSize(TPerforatorString str);

void PerforatorStringDispose(TPerforatorString str);

////////////////////////////////////////////////////////////////////////////////

#ifdef __cplusplus
}
#endif
