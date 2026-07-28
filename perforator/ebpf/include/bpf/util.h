#pragma once

#define ZERO(value) \
    __builtin_memset(&(value), 0, sizeof(value));

#define ARRAY_SIZE(x) \
    (sizeof(x) / sizeof((x)[0]))

#define CAT2(x, y) x ## y
#define CAT(x, y) CAT2(x, y)

// Forces LLVM to treat x as opaque so that subsequent masks (x & (MAX-1))
// are not optimized away. Required for 5.4 BPF verifier compatibility.
// See: tools/lib/bpf/, Cilium — same pattern.
#define BPF_VALUE_BARRIER(x) asm volatile("" : "+r"(x))

/*
 * Non-CO-RE variant of BPF_CORE_READ_USER that requires just an address to read
 * instead of struct field.
 *
 * To get correct type, cast the expression to the required type pointer
 *
 * As no CO-RE relocations are emitted, source types can be arbitrary and are
 * not restricted to kernel types only.
 *
 * TODO: Remove pragmas and use typeof_unqual when clang updates.
 *
 * Example: BPF_PROBE_READ_USER_FROM((MyStruct*)&array[index]);
 */
#define BPF_PROBE_READ_USER_FROM(src)                                                              \
    ({                                                                                             \
        _Pragma("clang diagnostic push");                                                          \
        _Pragma("clang diagnostic ignored \"-Wincompatible-pointer-types-discards-qualifiers\"");  \
        typeof(*(src)) __r;                                                                        \
        bpf_probe_read_user(&__r, sizeof(*(src)), (src));                                          \
        _Pragma("clang diagnostic pop");                                                           \
        __r;                                                                                       \
    })
