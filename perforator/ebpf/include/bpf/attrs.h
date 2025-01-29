#pragma once

#include "sections.h"


#define MAYBE_UNUSED __attribute__((__unused__))
#define ALWAYS_INLINE __attribute__((always_inline))
#define NOINLINE __attribute__((noinline))
#define PACKED __attribute__((__packed__))

#define SEC(NAME) __attribute__((section(NAME), used))
#define LICENSE(NAME) char __license[] SEC(BPF_SEC_LICENSE) = NAME;

#define BTF_EXPORT(typ) \
    struct CAT(btf_export_, __LINE__) { \
        typ field; \
    } CAT(unused_to_trigger_btf_generation, __LINE__) SEC("unused")

