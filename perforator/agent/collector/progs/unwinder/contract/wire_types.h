#pragma once

#include "language_ids.h"

#include <perforator/ebpf/include/bpf/types.h>

// Canonical perf-buffer ABI shared by the BPF producer and userspace consumers.

enum record_tag: u8 {
    RECORD_TAG_SAMPLE = 0,
    RECORD_TAG_NEW_PROCESS = 1,
};

enum sample_type: u32 {
    SAMPLE_TYPE_UNDEFINED = 0,
    SAMPLE_TYPE_PERF_EVENT = 1,
    SAMPLE_TYPE_KPROBE_FINISH_TASK_SWITCH = 2,
    SAMPLE_TYPE_TRACEPOINT_SIGNAL_DELIVER = 3,
    SAMPLE_TYPE_TRACEPOINT_SCHED_SWITCH = 4,
    SAMPLE_TYPE_UPROBE = 5,
};

enum language_id: u8 {
    LANGUAGE_PYTHON = PERFORATOR_LANGUAGE_PYTHON,
    LANGUAGE_PHP = PERFORATOR_LANGUAGE_PHP,
    LANGUAGE_JVM = PERFORATOR_LANGUAGE_JVM,
    LANGUAGE_LUA = PERFORATOR_LANGUAGE_LUA,
};

enum tls_variable_type: u8 {
    THREAD_LOCAL_UINT64_TYPE = 1,
    THREAD_LOCAL_STRING_TYPE = 2,
};

enum {
    PACKED_SAMPLE_MAX_DATA = 8192,
    MAX_THREAD_LOCAL_STRING_LENGTH = 128,
};

enum cgroup_limits {
    PARENT_CGROUP_MAX_LEVELS = 16,
};

enum cgroup_consts: u64 {
    END_OF_CGROUP_LIST = (u64)-1,
};

struct branch_record {
    u64 from;
    u64 to;
    // Preserve the 24-byte perf_branch_entry layout used by bpf_read_branch_records.
    u64 flags;
};

struct thread_local_string {
    u64 len;
    char string[MAX_THREAD_LOCAL_STRING_LENGTH];
};

union thread_local_variable {
    struct thread_local_string string;
    u64 number;
};

struct thread_local_variable_collect_result {
    u64 offset;
    enum tls_variable_type type;
    union thread_local_variable value;
};

struct perf_event_attr_subset {
    u32 type;
    u64 config;
};

struct perf_event_subset {
    struct perf_event_attr_subset attr;
    u64 id;
};

struct perf_event_value {
    u64 counter;
    u64 running;
    u64 enabled;
};

union sample_config {
    i32 sig;
    struct perf_event_subset perf_event;
    u64 ip;
};

struct record_new_process {
    enum record_tag tag;
    u32 pid;
    u64 starttime;
};

struct language_section_header {
    // Payload size in bytes, excluding this header.
    u16 byte_size;
    enum language_id language;
    u8 _pad0;
    u16 element_size;
    u8 _pad[2];
};

struct section_desc {
    // Byte offset from the start of packed_sample.data.
    u16 offset;
    u16 size;
};

struct record_sample_header {
    enum record_tag tag;
    enum sample_type sample_type;
    union sample_config sample_config;
    bool kthread;
    u16 cpu;
    u32 runtime;
    u64 collection_time;
    u8 thread_comm[16];
    u8 process_comm[16];
    u32 pid;
    u32 tid;
    u32 innermost_pidns_tid;
    u32 innermost_pidns_pid;
    u64 parent_cgroup;
    u64 starttime;
    struct perf_event_value value;
    u64 timedelta;
    struct section_desc kern_stack;
    struct section_desc user_stack;
    struct section_desc lbr;
    struct section_desc tls;
    struct section_desc cgroups;
    struct section_desc language_sections;
};
