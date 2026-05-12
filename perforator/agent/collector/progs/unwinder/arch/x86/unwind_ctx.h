#pragma once

#include <bpf/bpf.h>

struct unwind_context {
    u64 cfa; // rsp
    u64 fp; // rbp
    u64 ip; // rip
};
