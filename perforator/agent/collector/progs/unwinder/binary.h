#pragma once

#include <bpf/bpf.h>

typedef u64 binary_id;

enum binary_storage_params : u32 {
    MAX_BINARIES = 1024 * 1024
};