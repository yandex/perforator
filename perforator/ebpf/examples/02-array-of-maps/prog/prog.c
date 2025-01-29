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

struct gb_value {
    u8 buf[1024];
};

BTF_EXPORT(struct gb_value);

BPF_MAP_STRUCT(one_gigabyte, BPF_MAP_TYPE_ARRAY, u32, struct gb_value, 400 * 1024, 0);

struct {
    BPF_MAP_DEF_UINT(type, BPF_MAP_TYPE_ARRAY_OF_MAPS);
    BPF_MAP_DEF_UINT(key_size, sizeof(u32));
    BPF_MAP_DEF_UINT(max_entries, 1024);
    BPF_MAP_DEF_ARRAY(values, struct one_gigabyte);
} gigabytes SEC(BPF_SEC_BTF_MAPS);

SEC("tracepoint/sched/sched_switch")
int trace_sched_switch(struct trace_sched_switch_args* ctx) {
    BPF_PRINTK("Handling sched_switch %d -> %d\n", ctx->prev_pid, ctx->next_pid);
    return 0;
}

LICENSE("GPL")
