#pragma once

#include "cgroups.h"
#include "thread_local.h"
#include "lbr.h"
#include "python/types.h"
#include "thread_local.h"
#include "jvm/api.h"

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

    // Kprobes
    SAMPLE_TYPE_KPROBE_FINISH_TASK_SWITCH,

    // Tracepoints
    SAMPLE_TYPE_TRACEPOINT_SIGNAL_DELIVER,
    SAMPLE_TYPE_TRACEPOINT_SCHED_SWITCH,

    // Uprobes
    SAMPLE_TYPE_UPROBE,
};

struct interpreter_stack {
    struct interpreter_frame frames[PYTHON_MAX_STACK_DEPTH];
    u8 len;
};

struct perf_event_attr_subset {
    u32 type;
    u64 config;
};

struct perf_event_subset {
    struct perf_event_attr_subset attr;
    u64 id;
};

union sample_config {
    // For sample_type == SAMPLE_TYPE_TRACEPOINT_SIGNAL_DELIVER
    int sig;
    // For sample_type == SAMPLE_TYPE_PERF_EVENT
    struct perf_event_subset perf_event;
    // For sample_type == SAMPLE_TYPE_UPROBE
    u64 ip;
};

struct record_new_process {
    // Header of the perf event.
    enum record_tag tag;
    u32 pid;
    u64 starttime;
};

////////////////////////////////////////////////////////////////////////////////
// Packed sample format (variable-length wire format)
//
// All sections and elements are 8-byte aligned.
// Each section is described by a (u16 offset, u16 size) descriptor in the
// header. The parser can index any section directly without sequential walking.
////////////////////////////////////////////////////////////////////////////////

enum language_id : u8 {
    LANGUAGE_PYTHON = 0,
    LANGUAGE_PHP = 1,
    LANGUAGE_JVM = 2,
    LANGUAGE_LUA = 3,
};

// 8-byte aligned language section header (embedded in the language_sections data).
struct language_section_header {
    u16 byte_size;  // size of frame data that follows (not including this header)
    u8 language;    // language_id
    u8 _pad[5];
};

// JVM language section entry: correlates a user stack frame with a JVM method.
struct jvm_lang_entry {
    u16 user_stack_index;  // index into user stack IPs
    u8 _pad[6];
    u64 method_addr;
};

// Section descriptor: byte offset and byte size within packed data.
struct section_desc {
    u16 offset;
    u16 size;
};

// Fixed header for the packed sample format.
struct record_sample_header {
    enum record_tag tag;
    enum sample_type sample_type;
    union sample_config sample_config;

    bool kthread;
    u16 cpu;
    u32 runtime;
    u64 collection_time;

    u8 thread_comm[TASK_COMM_LEN];
    u8 process_comm[TASK_COMM_LEN];
    u32 pid;
    u32 tid;
    u32 innermost_pidns_tid;
    u32 innermost_pidns_pid;
    u64 parent_cgroup;
    u64 starttime;
    u64 value;
    u64 timedelta;

    u64 lua_vm_start_pc;
    u64 lua_vm_end_pc;

    // Section descriptors (offset + size in bytes, relative to start of data)
    struct section_desc kern_stack;
    struct section_desc user_stack;
    struct section_desc lbr;
    struct section_desc tls;
    struct section_desc cgroups;
    struct section_desc language_sections;  // all language sections contiguously
};

// Export types used by the Go parser (btf2go generates Go structs + GetSection
// accessor + compile-time offset assertions from these).
BTF_EXPORT(struct interpreter_stack);
BTF_EXPORT(struct jvm_stack);
BTF_EXPORT(struct tls_collect_result);
BTF_EXPORT(struct last_branch_records);
BTF_EXPORT(struct section_desc);
BTF_EXPORT(struct record_sample_header);
BTF_EXPORT(struct language_section_header);
BTF_EXPORT(enum language_id);

enum {
    PACKED_SAMPLE_MAX_DATA = 8192,
};

struct packed_sample {
    struct record_sample_header header;
    u8 data[PACKED_SAMPLE_MAX_DATA];
};

////////////////////////////////////////////////////////////////////////////////

#define BPF_PERFBUF_SUBMIT(map, var) \
    long res = bpf_perf_event_output(ctx, &map, BPF_F_CURRENT_CPU, var, sizeof(*var)); \
    if (res != 0) { \
        BPF_TRACE("bpf_perf_event_output failed: %ld\n", res); \
    } \

void submit_packed_sample(void* ctx, struct packed_sample* packed, u32 size) {
    packed->header.tag = RECORD_TAG_SAMPLE;
    if (size > sizeof(struct packed_sample)) {
        size = sizeof(struct packed_sample);
    }
#ifdef PERFORATOR_COMPAT_5_4
    long res = bpf_perf_event_output(ctx, &samples, BPF_F_CURRENT_CPU, packed, sizeof(struct packed_sample));
#else
    long res = bpf_perf_event_output(ctx, &samples, BPF_F_CURRENT_CPU, packed, size);
#endif
    if (res != 0) {
        BPF_TRACE("bpf_perf_event_output failed: %ld\n", res);
    }
}

void submit_new_process(void* ctx, struct record_new_process* rec) {
    rec->tag = RECORD_TAG_NEW_PROCESS;
    BPF_PERFBUF_SUBMIT(processes, rec);
}

////////////////////////////////////////////////////////////////////////////////
