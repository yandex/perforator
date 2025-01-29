#include <bpf/bpf.h>

// See /sys/kernel/debug/tracing/events/sched/sched_switch/format
struct trace_sched_switch_args {
    u64 ignore;
    u8 prev_comm[16];
    u32 prev_pid;
    u32 prev_prio;
    u64 prev_state;
    u8 next_comm[16];
    u32 next_pid;
    u32 next_prio;
};

static ALWAYS_INLINE void printsp() {
    u64 sp;
    // asm("mov %%r10, %0" : "=r" (sp));
    asm("%0 = r10" : "=r" (sp));
    BPF_PRINTK("sp=%llu\n", sp);
}

static NOINLINE void foo(int a, int b) {
    BPF_PRINTK("a=%a, b=%b\n", a, b);
    printsp();
}

SEC("tracepoint/sched/sched_switch")
int trace_sched_switch(struct trace_sched_switch_args* ctx) {
    BPF_PRINTK("Handling sched_switch %d -> %d\n", ctx->prev_pid, ctx->next_pid);
    printsp();
    foo(ctx->prev_pid, ctx->next_pid);
    return 0;
}

LICENSE("GPL")
