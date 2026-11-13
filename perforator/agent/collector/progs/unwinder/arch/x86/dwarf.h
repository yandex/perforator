#pragma once

#include <bpf/bpf.h>

#include "../../dwarf-fwd.h"

////////////////////////////////////////////////////////////////////////////////

enum {
    DWARF_CFI_UNKNOWN_REGISTER = 0xfffffffffffffffd,

    DWARF_CFI_FRAME_REGISTER_NUMBER = 6
};

enum dwarf_amd64_regno : u8 {
    DWARF_CFI_STACK_REGISTER_NUMBER = 7,
};

BTF_EXPORT(enum dwarf_amd64_regno);

static void ALWAYS_INLINE dwarf_cfi_context_init_next(struct unwind_context* ctx) {
    ctx->cfa = DWARF_CFI_UNKNOWN_REGISTER;
    ctx->fp = DWARF_CFI_UNKNOWN_REGISTER;
    ctx->ip = DWARF_CFI_UNKNOWN_REGISTER;
}

static ALWAYS_INLINE void dwarf_unwind_setup_userspace_registers(
    struct unwind_context* cfi,
    struct user_regs* regs
) {
    cfi->cfa = regs->rsp;
    cfi->fp = regs->rbp;
    cfi->ip = regs->rip;
}

////////////////////////////////////////////////////////////////////////////////

enum {
    // Registers a callee may change without describing it in its CFI. DWARF treats registers without a rule
    // as having the same value in the caller, and hand-written assembly relies on it: aesni_gcm_encrypt keeps
    // its CFA in rax and aesni_cbc_encrypt in r11 while calling helpers that preserve them.
    DWARF_GPRS_CALLER_SAVED =
        (1 << DWARF_RAX) | (1 << DWARF_RDX) | (1 << DWARF_RCX) | (1 << DWARF_RSI) | (1 << DWARF_RDI) |
        (1 << DWARF_R8) | (1 << DWARF_R9) | (1 << DWARF_R10) | (1 << DWARF_R11),
};

static ALWAYS_INLINE void dwarf_unwind_setup_userspace_gprs(struct unwind_gprs* gprs, struct user_regs* regs) {
    *gprs = regs->gprs;
}

// Callee-saved registers are restored from the stack by DW_CFA_offset rules, which the unwind table does not
// carry, so only caller-saved registers survive a DWARF step.
static ALWAYS_INLINE void dwarf_unwind_gprs_after_dwarf_step(struct unwind_gprs* gprs) {
    gprs->known &= DWARF_GPRS_CALLER_SAVED;
}

// Nothing is known about the registers of a frame unwound without CFI.
static ALWAYS_INLINE void dwarf_unwind_gprs_forget(struct unwind_gprs* gprs) {
    gprs->known = 0;
}

static ALWAYS_INLINE bool dwarf_unwind_gprs_get(struct unwind_gprs* gprs, u8 regno, u64* value) {
    if (regno >= DWARF_GPR_COUNT || !(gprs->known & (1 << regno))) {
        return false;
    }
    *value = gprs->regs[regno];
    return true;
}

////////////////////////////////////////////////////////////////////////////////

ALWAYS_INLINE bool dwarf_cfi_eval_ra(
    struct unwind_context* prev,
    struct unwind_context* next,
    struct ra_unwind_rule* rule
) {
    u64 address = next->cfa - 8;
    return read_return_address((void*)address, &next->ip);
}

// Canonical function prologue and epilogue:
// foo:
//      push %rbp
//      mov %rsp, %rbp
//      ...
//      mov %rbp, %rsp
//      pop %rbp
//      ret
//
// Stack layout:
// rsp0    -> [....]
// rsp0-8  -> [ra0 ]
// rsp0-16 -> [rbp0]
// ...
// rsp1    -> [....]
// rsp1-8  -> [ra1 ]
// rsp1-16 -> [rbp1]
// ...
// rsp2    -> [....]
// rsp2-8  -> [ra2 ]
// rsp2-16 -> [rbp2]
static NOINLINE enum dwarf_unwind_step_result dwarf_unwind_step_fp(struct unwind_context* cfi, u32* framepointers) {
    if (cfi == NULL || framepointers == NULL) {
        return DWARF_UNWIND_STEP_FAILED;
    }

    if (cfi->fp == 0) {
        return DWARF_UNWIND_STEP_FINISHED;
    }
    if (cfi->fp == DWARF_CFI_UNKNOWN_REGISTER) {
        return DWARF_UNWIND_STEP_FAILED;
    }

    (*framepointers)++;
    if (!read_return_address((void*)(cfi->fp + 8), &cfi->ip)) {
        metric_increment(METRIC_FP_ERROR_READ_RETURNADDRESS_COUNT);
        return DWARF_UNWIND_STEP_FAILED;
    }

    // We need to restore rsp in order to support mixed dwarf and frame pointers unwinding.
    // Previous rsp is equal to current rbp + 2*sizeof(register): look at the stack layout.
    cfi->cfa = cfi->fp + 16;

    u64 prev_rbp = 0;
    int err = bpf_probe_read_user(&prev_rbp, sizeof(prev_rbp), (void*)cfi->fp);
    if (err != 0) {
        DWARF_TRACE("fp: bpf_probe_read_user failed: %d\n", err);
        metric_increment(METRIC_FP_ERROR_READ_BASEPOINTER_COUNT);
        return DWARF_UNWIND_STEP_FAILED;
    }
    cfi->fp = prev_rbp;

    return DWARF_UNWIND_STEP_CONTINUE;
}
