#pragma once

#include <bpf/attrs.h>
#include <bpf/core.h>
#include <bpf/types.h>

#include "../../core.h"
#include "../../linux.h"
#include "../../task.h"

////////////////////////////////////////////////////////////////////////////////

struct user_regs {
    u64 sp;
    u64 fp;
    u64 pc;
    u64 lr;
};

// CFA based on registers other than sp and fp is not supported on arm64.
struct unwind_gprs {
    u16 known;
};

static ALWAYS_INLINE u64 regs_get_current_instruction(struct user_regs* regs) {
    return regs->pc;
}

// FIXME: this is suboptimal, we are using only part of user-spaced regs
struct pt_regs___kernel {
    u64 regs[31];
    u64 sp;
    u64 pc;
};

////////////////////////////////////////////////////////////////////////////////

// See https://www.kernel.org/doc/Documentation/arm64/memory.txt
ALWAYS_INLINE bool is_kernel_pc(u64 pc) {
    return pc > 0xffff000000000000;
}

// See https://github.com/iovisor/bcc/issues/2073#issuecomment-446844179
// And https://elixir.bootlin.com/linux/v5.4.254/source/arch/arm/include/asm/ptrace.h#L16
// And https://elixir.bootlin.com/linux/v5.4.254/source/arch/arm/include/asm/processor.h#L98
static NOINLINE bool extract_saved_userspace_registers(struct user_regs* regs) {
    struct pt_regs___kernel* kregs = 0;
    if (bpf_core_enum_value_exists(enum bpf_func_id, BPF_FUNC_task_pt_regs)) {
        struct task_struct* task = bpf_get_current_task_btf();
        if (task == NULL) {
            return false;
        }

        kregs = (void*)bpf_task_pt_regs(task);
        if (kregs == 0) {
            return false;
        }
    } else {
        struct task_struct* task = get_current_task();
        if (task == NULL) {
            return false;
        }

        void* ptr = BPF_CORE_READ(task, stack);
        if (ptr == NULL) {
            return false;
        }

        ptr += THREAD_START_SP;
        kregs = (void*)((struct user_pt_regs*)(ptr) - 1);
    }

    regs->pc = BPF_CORE_READ(kregs, pc);
    regs->fp = BPF_CORE_READ(kregs, ARM_FP);
    regs->sp = BPF_CORE_READ(kregs, sp);
    regs->lr = BPF_CORE_READ(kregs, ARM_LR);

    return true;
}

// FIXME: bpf_perf_event_data contains only user_pt_regs, not full regs
static NOINLINE bool find_task_userspace_registers_bpf(struct user_pt_regs* kregs, struct user_regs* uregs) {
    if (is_kernel_pc(kregs->pc)) {
        return extract_saved_userspace_registers(uregs);
    }

    uregs->sp = kregs->sp;
    uregs->fp = kregs->ARM_FP;
    uregs->pc = kregs->pc;
    uregs->lr = kregs->ARM_LR;

    return true;
}

static ALWAYS_INLINE bool find_task_userspace_registers(struct pt_regs* kregs, struct user_regs* uregs) {
    return find_task_userspace_registers_bpf(&kregs->user_regs, uregs);
}
