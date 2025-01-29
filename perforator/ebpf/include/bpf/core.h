#pragma once

#include <contrib/libs/libbpf/src/bpf_core_read.h>

// Get real offset of the field in the structure.
#define BPF_CORE_FIELD_OFFSET(field) \
    __builtin_preserve_field_info(field, BPF_FIELD_BYTE_OFFSET)

// Get enum value by name.
#define BPF_CORE_ENUM_VALUE(enum_type, enum_value) \
    __builtin_preserve_enum_value(*(typeof(enum_type)*)enum_value, 1)

// Check if field exists
#define BPF_CORE_FIELD_EXISTS(field...) \
    __builtin_preserve_field_info(field, BPF_FIELD_EXISTS)
