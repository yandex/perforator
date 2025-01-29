#pragma once

#include "binary.h"

#include <bpf/types.h>

enum unwind_type : u8 {
    UNWIND_TYPE_DISABLED = 0,
    UNWIND_TYPE_FP = 1,
    UNWIND_TYPE_DWARF = 2,
};

struct process_info {
    enum unwind_type unwind_type;
    binary_id main_binary_id;
};
