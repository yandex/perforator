#pragma once

#include <util/system/types.h>

#ifdef __cplusplus
extern "C" {
#endif

const char* ConvertDWARFToGSYM(const char* input, const char* output, ui32 convertNumThreads);

#ifdef __cplusplus
}
#endif
