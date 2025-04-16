#pragma once

#include "binary.h"

#include <bpf/types.h>

enum unwind_type : u8 {
    UNWIND_TYPE_DISABLED = 0,
    UNWIND_TYPE_FP = 1,
    UNWIND_TYPE_DWARF = 2,
};

enum special_binary_type : u8 {
    SPECIAL_BINARY_TYPE_NONE = 0,
    SPECIAL_BINARY_TYPE_PYTHON_INTERPRETER = 1,
    SPECIAL_BINARY_TYPE_PTHREAD_GLIBC = 2,
};

struct special_binary {
    binary_id id;
    u64 start_address;
    enum special_binary_type type;
};

struct process_info {
    enum unwind_type unwind_type;
    binary_id main_binary_id;

    struct special_binary pthread_binary;

    // Interpreter binary ID may be different from main binary ID.
    // For example, if CPython is dynamically linked
    // into the main binary.
    struct special_binary interpreter_binary;
};
