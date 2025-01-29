#pragma once

#include <bpf/attrs.h>
#include <bpf/core.h>
#include <bpf/types.h>

#include "../../core.h"
#include "../../linux.h"
#include "../../task.h"

////////////////////////////////////////////////////////////////////////////////

struct user_regs {
    u64 rsp;
    u64 rbp;
    u64 rip;
};

struct pt_regs___kernel {
    u64 ip;
    u64 sp;
    u64 bp;
};

////////////////////////////////////////////////////////////////////////////////

// See https://www.kernel.org/doc/Documentation/x86/x86_64/mm.txt
ALWAYS_INLINE bool is_kernel_ip(u64 ip) {
    return ip > 0xff00000000000000ull;
}

// See https://github.com/iovisor/bcc/issues/2073#issuecomment-446844179
// And https://elixir.bootlin.com/linux/v5.4.254/source/arch/x86/include/asm/processor.h#L813
// And https://elixir.bootlin.com/linux/v5.4.254/source/arch/x86/include/asm/ptrace.h#L56
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

        ptr += THREAD_SIZE - TOP_OF_KERNEL_STACK_PADDING;
        kregs = (void*)(((struct pt_regs*)ptr) - 1);
    }

    regs->rip = BPF_CORE_READ(kregs, ip);
    regs->rbp = BPF_CORE_READ(kregs, bp);
    regs->rsp = BPF_CORE_READ(kregs, sp);

    return true;
}

static NOINLINE bool find_task_userspace_registers(struct pt_regs* kregs, struct user_regs* uregs) {
    if (is_kernel_ip(kregs->rip)) {
        return extract_saved_userspace_registers(uregs);
    }

    uregs->rsp = kregs->rsp;
    uregs->rbp = kregs->rbp;
    uregs->rip = kregs->rip;

    return true;
}

////////////////////////////////////////////////////////////////////////////////
