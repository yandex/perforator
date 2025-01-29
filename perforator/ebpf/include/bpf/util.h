#pragma once

#define ZERO(value) \
    __builtin_memset(&(value), 0, sizeof(value));

#define ARRAY_SIZE(x) \
    (sizeof(x) / sizeof((x)[0]))

#define CAT2(x, y) x ## y
#define CAT(x, y) CAT2(x, y)
