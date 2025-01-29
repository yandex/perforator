#pragma once

#include "cgroups.h"
#include "thread_local.h"
#include "lbr.h"
#include "python.h"
#include "thread_local.h"

#include <linux/perf_event.h>
#include <bpf/bpf.h>

////////////////////////////////////////////////////////////////////////////////

BPF_MAP(samples, BPF_MAP_TYPE_PERF_EVENT_ARRAY, u32, u32, 0);
BPF_MAP(processes, BPF_MAP_TYPE_PERF_EVENT_ARRAY, u32, u32, 0);

////////////////////////////////////////////////////////////////////////////////

enum record_tag : u8 {
    RECORD_TAG_SAMPLE = 0,
    RECORD_TAG_NEW_PROCESS = 1,
};

enum sample_type : u32 {
    SAMPLE_TYPE_UNDEFINED = 0,

    // Perf events
    SAMPLE_TYPE_PERF_EVENT,

    // Krpobes
    SAMPLE_TYPE_KPROBE_FINISH_TASK_SWITCH,

    // Tracpoints
    SAMPLE_TYPE_TRACEPOINT_SIGNAL_DELIVER,
    SAMPLE_TYPE_TRACEPOINT_SCHED_SWITCH,
};

struct record_sample {
    // Header of the perf event.
    enum record_tag tag;

    // Where this sample come from.
    enum sample_type sample_type;

    // Auxillary info specific to the concrete sample_type.
    // ID of the perf_event if sample_type == SAMPLE_TYPE_PERF_EVENT.
    // Signal number if sample_type == SAMPLE_TYPE_TRACEPOINT_SIGNAL_DELIVER.
    u64 sample_config;

    // Is sample task a kernel thread.
    bool kthread;

    // Index of the CPU this event was triggered on.
    u16 cpu;

    // Number of nanoseconds the eBPF program was running
    // in terms of bpf_ktime_get_ns (clock_gettime(CLOCK_MONOTONIC))
    u32 runtime;

    u8 thread_comm[TASK_COMM_LEN];
    u8 process_comm[TASK_COMM_LEN];
    u32 pid;
    u32 tid;
    u64 parent_cgroup;
    // All cgroups starting from innermost and up to (but not including) parent.
    // Terminated by -1 when too short.
    u64 cgroups_hierarchy[PARENT_CGROUP_MAX_LEVELS];
    u64 starttime;
    u64 kernstack[PERF_MAX_STACK_DEPTH];
    u64 userstack[PERF_MAX_STACK_DEPTH];

    u8 python_stack_len;
    struct python_frame python_stack[PYTHON_MAX_STACK_DEPTH];

    struct tls_collect_result tls_values;

    struct last_branch_records lbr_values;

    // Sample value (e.g. cycles).
    u64 value;

    // Number of nanoseconds since last thread sample. 0 for the first sample.
    u64 timedelta;
};

struct record_new_process {
    // Header of the perf event.
    enum record_tag tag;
    u32 pid;
    u64 starttime;
};

////////////////////////////////////////////////////////////////////////////////

#define BPF_PERFBUF_SUBMIT(map, var) \
    long res = bpf_perf_event_output(ctx, &map, BPF_F_CURRENT_CPU, var, sizeof(*var)); \
    if (res != 0) { \
        BPF_TRACE("bpf_perf_event_output failed: %ld\n", res); \
    } \

void submit_sample(void* ctx, struct record_sample* rec) {
    rec->tag = RECORD_TAG_SAMPLE;
    BPF_PERFBUF_SUBMIT(samples, rec);
}

void submit_new_process(void* ctx, struct record_new_process* rec) {
    rec->tag = RECORD_TAG_NEW_PROCESS;
    BPF_PERFBUF_SUBMIT(processes, rec);
}

////////////////////////////////////////////////////////////////////////////////
