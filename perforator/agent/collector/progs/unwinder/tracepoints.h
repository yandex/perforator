#pragma once

#include <bpf/types.h>

struct trace_event_header {
    short unsigned int type;
    unsigned char flags;
    unsigned char preempt_count;
    int pid;
};

struct trace_event_sched_switch {
    u64 ignore;
    u8 prev_comm[16];
    u32 prev_pid;
    u32 prev_prio;
    u64 prev_state;
    u8 next_comm[16];
    u32 next_pid;
    u32 next_prio;
};

struct trace_event_sched_stat_runtime {
    struct trace_event_header hdr;
    char comm[16];
    int pid;
    u64 runtime;
    u64 vruntime;
};

struct trace_event_signal_deliver {
    struct trace_event_header hdr;
    int sig;
    int errno;
    int code;
    long unsigned int sa_handler;
    long unsigned int sa_flags;
};
