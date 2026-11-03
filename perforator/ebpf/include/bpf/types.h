#pragma once

#include <linux/types.h>

#ifdef __cplusplus
    #include <stdint.h>
#endif

typedef __u8 u8;
typedef __u16 u16;
typedef __u32 u32;
typedef __u64 u64;

typedef __s8 i8;
typedef __s16 i16;
typedef __s32 i32;

#ifdef __cplusplus
    // Match the userspace i64 alias from util/system/types.h.
    typedef int64_t i64;
#else
    typedef __s64 i64;
    typedef _Bool bool;
    #define false 0
    #define true 1
#endif
