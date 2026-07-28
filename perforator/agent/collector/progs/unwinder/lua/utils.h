#pragma once

/*
 * Non-CO-RE variant of BPF_CORE_READ_USER that requires just an address to read
 * instead of struct field.
 *
 * To get correct type, cast the expression to the required type pointer
 *
 * As no CO-RE relocations are emitted, source types can be arbitrary and are
 * not restricted to kernel types only.
 */
// clang-format off
#define BPF_PROBE_READ_USER_FROM(src)                                                           \
    ({                                                                                             \
        _Pragma("clang diagnostic push");                                                          \
        _Pragma("clang diagnostic ignored \"-Wincompatible-pointer-types-discards-qualifiers\"");  \
        typeof(*(src)) __r;                                                                        \
        bpf_probe_read_user(&__r, sizeof(*(src)), (src));                                          \
        _Pragma("clang diagnostic pop");                                                           \
        __r;                                                                                       \
    })
// clang-format on
