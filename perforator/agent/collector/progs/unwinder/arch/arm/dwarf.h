#pragma once

#include <bpf/bpf.h>

#include "../../dwarf-fwd.h"

////////////////////////////////////////////////////////////////////////////////

enum {
    DWARF_CFI_UNKNOWN_REGISTER = 0xfffffffffffffffd,

    DWARF_CFI_FRAME_REGISTER_NUMBER = 29
};

enum dwarf_aarch64_regno {
    DWARF_CFI_STACK_REGISTER_NUMBER = 31,
};

BTF_EXPORT(enum dwarf_aarch64_regno);

static void ALWAYS_INLINE dwarf_cfi_context_init_next(struct unwind_context* ctx) {
    ctx->cfa = DWARF_CFI_UNKNOWN_REGISTER;
    ctx->fp = DWARF_CFI_UNKNOWN_REGISTER;
    ctx->ip = DWARF_CFI_UNKNOWN_REGISTER;
    ctx->lr = DWARF_CFI_UNKNOWN_REGISTER;
}

static ALWAYS_INLINE void dwarf_unwind_setup_userspace_registers(
    struct unwind_context* cfi,
    struct user_regs* regs
) {
    cfi->cfa = regs->sp;
    cfi->fp = regs->fp;
    cfi->ip = regs->pc;

    cfi->lr = regs->lr;
}

////////////////////////////////////////////////////////////////////////////////

// FIXME: we are fully dependendent on Link register here, but it might be invalidated
// See: https://github.com/ARM-software/abi-aa/blob/main/aadwarf64/aadwarf64.rst#note-8
// Currently not a single observed binary has this DWARF expression
ALWAYS_INLINE bool dwarf_cfi_eval_ra(
    struct unwind_context* prev,
    struct unwind_context* next,
    struct ra_unwind_rule* rule
) {
    if (rule->offset == DWARF_UNWIND_CFA_RULE_UNDEFINED) {
        if (prev->lr == 0 || prev->lr == DWARF_CFI_UNKNOWN_REGISTER) {
            DWARF_TRACE("RA rule unavailable and LR invalid");
            return false;
        }

        DWARF_TRACE("RA rule unavailable, using LR as IP");
        next->ip = prev->lr;

        if (next->fp != DWARF_CFI_UNKNOWN_REGISTER) {
            if (!read_return_address((void*)(next->fp + 8), &next->lr)) {
                next->lr = DWARF_CFI_UNKNOWN_REGISTER;
            }
        }

        return true;
    }

    u64 address = next->cfa + rule->offset;
    if (!read_return_address((void*)address, &next->lr)) {
        return false;
    }
    next->ip = next->lr;

    return true;
}

////////////////////////////////////////////////////////////////////////////////

// There is no canonical prolouge/epilogue:
// See: https://www.codalogic.com/blog/2022/10/20/Aarch64-Stack-Frames-Again
//
// Clang function prologue and epilogue:
// foo:
//      sub sp, sp, #size
//      stp x29, x30, [sp, #(size - 16)]
//      add x29, sp, #(size - 16)
//      ...
//      ldp x29, x30, [sp, #(size - 16)]
//      add sp, sp, #size
//      ret
//
// Stack layout:
// |                     |
// +---------------------+
// |          lr         |
// +---------------------+
// |    original fp      | <- fp
// +---------------------+
// |                     |
// |                     |
// |     ...space...     |
// |                     |
// |                     | <- sp
// +---------------------+
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

    cfi->ip = cfi->lr == DWARF_CFI_UNKNOWN_REGISTER ? 0 : cfi->lr;
    if (!read_return_address((void*)(cfi->fp + 8), &cfi->lr)) {
        metric_increment(METRIC_FP_ERROR_READ_RETURNADDRESS_COUNT);
        return DWARF_UNWIND_STEP_FAILED;
    }

    cfi->cfa = cfi->fp + 16;

    u64 prev_fp = 0;
    int err = bpf_probe_read_user(&prev_fp, sizeof(prev_fp), (void*)cfi->fp);
    if (err != 0) {
        DWARF_TRACE("fp: bpf_probe_read_user failed: %d\n", err);
        metric_increment(METRIC_FP_ERROR_READ_BASEPOINTER_COUNT);
        return DWARF_UNWIND_STEP_FAILED;
    }
    cfi->fp = prev_fp;

    return DWARF_UNWIND_STEP_CONTINUE;
}
