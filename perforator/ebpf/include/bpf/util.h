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
