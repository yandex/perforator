#pragma once

#include <bpf/bpf.h>

#define TRACE_DWARF_UNWINDING
#ifdef TRACE_DWARF_UNWINDING
#define DWARF_TRACE(...) BPF_TRACE(__VA_ARGS__)
#else
#define DWARF_TRACE(...)
#endif

enum unwind_rule_kind : u8 {
    UNWIND_RULE_UNSUPPORTED = 0,
    UNWIND_RULE_CFA_MINUS_8 = 1,
    UNWIND_RULE_CFA_PLUS_OFFSET = 2,
    UNWIND_RULE_REGISTER_OFFSET = 3,
    // CFA = *(reg + offset): gcc stack realignment through DRAP.
    UNWIND_RULE_REGISTER_DEREF_OFFSET = 4,
    UNWIND_RULE_PLT_SECTION = 5,
    UNWIND_RULE_CONSTANT = 6,
    UNWIND_RULE_STOP = 7,
    // CFA = *(reg + offset) + 8: hand-written openssl perlasm.
    // The bias is a part of the kind: there is no room for it in struct cfa_unwind_rule.
    UNWIND_RULE_REGISTER_DEREF_OFFSET_PLUS_8 = 8,
};

struct  __attribute__((packed)) cfa_unwind_rule {
    enum unwind_rule_kind kind;
    u8 regno;
    i32 offset;
};

enum dwarf_cfa_well_known {
    DWARF_UNWIND_CFA_RULE_UNDEFINED = 0x7f,
    // RBP is saved at the address in RBP: deref(RBP), gcc stack realignment through DRAP.
    DWARF_UNWIND_RBP_RULE_AT_RBP = 0x7e,
};

BTF_EXPORT(enum dwarf_cfa_well_known);

struct rbp_unwind_rule {
    // Offset from the CFA to read saved value of RBP from: deref(CFA+offset),
    // or one of the special values from enum dwarf_cfa_well_known.
    i8 offset;
};

struct  __attribute__((packed)) ra_unwind_rule {
    // Same as rbp rule
    // Only deref(CFA+offset) is supported
    i8 offset;
};

struct __attribute__((packed)) unwind_rule {
    struct cfa_unwind_rule cfa;
    struct rbp_unwind_rule rbp;
    struct ra_unwind_rule ra;
};

enum dwarf_unwind_step_result {
    DWARF_UNWIND_STEP_CONTINUE = 0,
    DWARF_UNWIND_STEP_FINISHED = 1,
    DWARF_UNWIND_STEP_FAILED = 2,
};

