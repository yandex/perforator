#pragma once

#include <bpf/attrs.h>
#include <bpf/core.h>
#include <bpf/types.h>

#include "../../core.h"
#include "../../linux.h"
#include "../../task.h"

////////////////////////////////////////////////////////////////////////////////

// DWARF register numbers of the general purpose registers, see the x86-64 psABI, figure 3.36.
enum dwarf_amd64_gpr : u8 {
    DWARF_RAX = 0,
    DWARF_RDX = 1,
    DWARF_RCX = 2,
    DWARF_RBX = 3,
    DWARF_RSI = 4,
    DWARF_RDI = 5,
    DWARF_RBP = 6,
    DWARF_RSP = 7,
    DWARF_R8 = 8,
    DWARF_R9 = 9,
    DWARF_R10 = 10,
    DWARF_R11 = 11,
    DWARF_R12 = 12,
    DWARF_R13 = 13,
    DWARF_R14 = 14,
    DWARF_R15 = 15,
    DWARF_GPR_COUNT = 16,
};

// General purpose registers indexed by DWARF register number. Hand-written assembly (openssl perlasm) and
// gcc stack realignment use them as the CFA base.
struct unwind_gprs {
    u64 regs[DWARF_GPR_COUNT];
    // Bit N is set if regs[N] holds the value of register N in the frame being unwound.
    u16 known;
};

enum {
    // Overwritten on syscall entry: rax holds the syscall number, the syscall instruction stores rip in rcx
    // and rflags in r11.
    DWARF_GPRS_SYSCALL_CLOBBERED = (1 << DWARF_RAX) | (1 << DWARF_RCX) | (1 << DWARF_R11),
};

struct user_regs {
    u64 rsp;
    u64 rbp;
    u64 rip;
    u64 rdi;
    u64 rdx;
    u64 r14;

    struct unwind_gprs gprs;
};

static ALWAYS_INLINE u64 regs_get_current_instruction(struct user_regs* regs) {
    return regs->rip;
}

struct pt_regs___kernel {
    u64 ip;
    u64 sp;
    u64 bp;
    u64 di;
    u64 dx;
    u64 r14;

    u64 ax;
    u64 cx;
    u64 bx;
    u64 si;
    u64 r8;
    u64 r9;
    u64 r10;
    u64 r11;
    u64 r12;
    u64 r13;
    u64 r15;
    u64 orig_ax;
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
    regs->rdi = BPF_CORE_READ(kregs, di);
    regs->rdx = BPF_CORE_READ(kregs, dx);
    regs->r14 = BPF_CORE_READ(kregs, r14);

    struct unwind_gprs* gprs = &regs->gprs;
    gprs->regs[DWARF_RAX] = BPF_CORE_READ(kregs, ax);
    gprs->regs[DWARF_RDX] = regs->rdx;
    gprs->regs[DWARF_RCX] = BPF_CORE_READ(kregs, cx);
    gprs->regs[DWARF_RBX] = BPF_CORE_READ(kregs, bx);
    gprs->regs[DWARF_RSI] = BPF_CORE_READ(kregs, si);
    gprs->regs[DWARF_RDI] = regs->rdi;
    gprs->regs[DWARF_RBP] = regs->rbp;
    gprs->regs[DWARF_RSP] = regs->rsp;
    gprs->regs[DWARF_R8] = BPF_CORE_READ(kregs, r8);
    gprs->regs[DWARF_R9] = BPF_CORE_READ(kregs, r9);
    gprs->regs[DWARF_R10] = BPF_CORE_READ(kregs, r10);
    gprs->regs[DWARF_R11] = BPF_CORE_READ(kregs, r11);
    gprs->regs[DWARF_R12] = BPF_CORE_READ(kregs, r12);
    gprs->regs[DWARF_R13] = BPF_CORE_READ(kregs, r13);
    gprs->regs[DWARF_R14] = regs->r14;
    gprs->regs[DWARF_R15] = BPF_CORE_READ(kregs, r15);
    gprs->known = 0xffff;

    // orig_ax is the syscall number inside a syscall and negative for interrupts. Exceptions with an error
    // code (page faults) also leave it non-negative; treating them as syscalls only loses these registers.
    if ((i64)BPF_CORE_READ(kregs, orig_ax) >= 0) {
        gprs->known &= ~DWARF_GPRS_SYSCALL_CLOBBERED;
    }

    return true;
}

static NOINLINE bool find_task_userspace_registers(struct pt_regs* kregs, struct user_regs* uregs) {
    if (is_kernel_ip(kregs->rip)) {
        return extract_saved_userspace_registers(uregs);
    }

    uregs->rsp = kregs->rsp;
    uregs->rbp = kregs->rbp;
    uregs->rip = kregs->rip;
    uregs->rdi = kregs->rdi;
    uregs->rdx = kregs->rdx;
    uregs->r14 = kregs->r14;

    // Interrupted in userspace: every register holds the value of the leaf frame.
    struct unwind_gprs* gprs = &uregs->gprs;
    gprs->regs[DWARF_RAX] = kregs->rax;
    gprs->regs[DWARF_RDX] = kregs->rdx;
    gprs->regs[DWARF_RCX] = kregs->rcx;
    gprs->regs[DWARF_RBX] = kregs->rbx;
    gprs->regs[DWARF_RSI] = kregs->rsi;
    gprs->regs[DWARF_RDI] = kregs->rdi;
    gprs->regs[DWARF_RBP] = kregs->rbp;
    gprs->regs[DWARF_RSP] = kregs->rsp;
    gprs->regs[DWARF_R8] = kregs->r8;
    gprs->regs[DWARF_R9] = kregs->r9;
    gprs->regs[DWARF_R10] = kregs->r10;
    gprs->regs[DWARF_R11] = kregs->r11;
    gprs->regs[DWARF_R12] = kregs->r12;
    gprs->regs[DWARF_R13] = kregs->r13;
    gprs->regs[DWARF_R14] = kregs->r14;
    gprs->regs[DWARF_R15] = kregs->r15;
    gprs->known = 0xffff;

    return true;
}

static ALWAYS_INLINE bool find_task_userspace_registers_bpf(struct pt_regs* kregs, struct user_regs* uregs) {
    return find_task_userspace_registers(kregs, uregs);
}

////////////////////////////////////////////////////////////////////////////////
