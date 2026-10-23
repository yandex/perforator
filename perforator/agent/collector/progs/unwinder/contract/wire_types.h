#pragma once

#include "language_ids.h"

#ifdef __cplusplus
    #include <linux/types.h>
#else
    #include <perforator/ebpf/include/bpf/types.h>
#endif

// Canonical perf-buffer ABI shared by the BPF producer and userspace consumers.

enum record_tag: __u8 {
    RECORD_TAG_SAMPLE = 0,
    RECORD_TAG_NEW_PROCESS = 1,
};

enum sample_type: __u32 {
    SAMPLE_TYPE_UNDEFINED = 0,
    SAMPLE_TYPE_PERF_EVENT = 1,
    SAMPLE_TYPE_KPROBE_FINISH_TASK_SWITCH = 2,
    SAMPLE_TYPE_TRACEPOINT_SIGNAL_DELIVER = 3,
    SAMPLE_TYPE_TRACEPOINT_SCHED_SWITCH = 4,
    SAMPLE_TYPE_UPROBE = 5,
};

enum language_id: __u8 {
    LANGUAGE_PYTHON = PERFORATOR_LANGUAGE_PYTHON,
    LANGUAGE_PHP = PERFORATOR_LANGUAGE_PHP,
    LANGUAGE_JVM = PERFORATOR_LANGUAGE_JVM,
    LANGUAGE_LUA = PERFORATOR_LANGUAGE_LUA,
};

enum language_payload_kind: __u8 {
    // Interpreter frames; the native parser keeps their payload opaque.
    LANGUAGE_PAYLOAD_INTERPRETER_FRAMES = 1,
    // Entries annotate native stack positions and therefore have to be
    // interpreted by the native stack builder (currently JVM only).
    LANGUAGE_PAYLOAD_NATIVE_ANNOTATIONS = 2,
};

enum tls_variable_type: __u8 {
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

enum cgroup_consts: __u64 {
    END_OF_CGROUP_LIST = (__u64)-1,
};

struct branch_record {
    __u64 from;
    __u64 to;
    // Preserve the 24-byte perf_branch_entry layout used by bpf_read_branch_records.
    __u64 flags;
};

struct thread_local_string {
    __u64 len;
    char string[MAX_THREAD_LOCAL_STRING_LENGTH];
};

union thread_local_variable {
    struct thread_local_string string;
    __u64 number;
};

struct thread_local_variable_collect_result {
    __u64 offset;
    enum tls_variable_type type;
    union thread_local_variable value;
};

struct perf_event_attr_subset {
    __u32 type;
    __u64 config;
};

struct perf_event_subset {
    struct perf_event_attr_subset attr;
    __u64 id;
};

union sample_config {
    __s32 sig;
    struct perf_event_subset perf_event;
    __u64 ip;
};

struct record_new_process {
    enum record_tag tag;
    __u32 pid;
    __u64 starttime;
};

struct language_section_header {
    // Payload size in bytes, excluding this header.
    __u16 byte_size;
    enum language_id language;
    enum language_payload_kind payload_kind;
    __u16 element_size;
    __u8 _pad[2];
};

struct jvm_lang_entry {
    __u16 user_stack_index;
    __u8 _pad[6];
    __u64 method_addr;
};

struct section_desc {
    // Byte offset from the start of packed_sample.data.
    __u16 offset;
    __u16 size;
};

struct record_sample_header {
    enum record_tag tag;
    enum sample_type sample_type;
    union sample_config sample_config;
    bool kthread;
    __u16 cpu;
    __u32 runtime;
    __u64 collection_time;
    __u8 thread_comm[16];
    __u8 process_comm[16];
    __u32 pid;
    __u32 tid;
    __u32 innermost_pidns_tid;
    __u32 innermost_pidns_pid;
    __u64 parent_cgroup;
    __u64 starttime;
    __u64 value;
    __u64 timedelta;
    struct section_desc kern_stack;
    struct section_desc user_stack;
    struct section_desc lbr;
    struct section_desc tls;
    struct section_desc cgroups;
    struct section_desc language_sections;
};
