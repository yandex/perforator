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
    UNWIND_RULE_REGISTER_DERREF_OFFSET = 4,
    UNWIND_RULE_PLT_SECTION = 5,
    UNWIND_RULE_CONSTANT = 6,
};

struct  __attribute__((packed)) cfa_unwind_rule {
    enum unwind_rule_kind kind;
    u8 regno;
    i32 offset;
};

enum dwarf_cfa_well_known {
    DWARF_UNWIND_CFA_RULE_UNDEFINED = 0x7f,
};

BTF_EXPORT(enum dwarf_cfa_well_known);

struct rbp_unwind_rule {
    // Offset from the CFA to read saved value of RBP from.
    // Now we support only one unwind rule for rbp: deref(CFA+offset).
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

ALWAYS_INLINE bool read_return_address(void* location, u64* ra) {
    DWARF_TRACE("read_return_address: read RA from %p\n", location);
    int err = bpf_probe_read_user(ra, sizeof(*ra), location);
    if (err != 0) {
        DWARF_TRACE("read_return_address: bpf_probe_read_user failed: %d\n", err);
        return false;
    }

    // Return address points to the next instruction after the call, not the call itself.
    // So we need to adjust return address to point to the real call instruction.
    *ra -= 1;

    return true;
}
