#pragma once

#include "language_ids.h"

// Canonical perf-buffer ABI shared by the BPF producer and the native
// userspace consumer. The producer uses the defaults below; the C++ wrapper
// overrides primitive and field-name macros before including this file.
#ifndef PERFORATOR_WIRE_U8
    #define PERFORATOR_WIRE_U8 u8
    #define PERFORATOR_WIRE_U16 u16
    #define PERFORATOR_WIRE_U32 u32
    #define PERFORATOR_WIRE_U64 u64
    #define PERFORATOR_WIRE_I32 i32
    #define PERFORATOR_WIRE_FIELD(bpf_name, host_name) bpf_name
    #define PERFORATOR_WIRE_ENUM(bpf_name, host_name) bpf_name
    #define PERFORATOR_WIRE_DEFAULT_MACROS
#endif

enum record_tag: PERFORATOR_WIRE_U8 {
    PERFORATOR_WIRE_ENUM(RECORD_TAG_SAMPLE, Sample) = 0,
    PERFORATOR_WIRE_ENUM(RECORD_TAG_NEW_PROCESS, NewProcess) = 1,
};

enum sample_type: PERFORATOR_WIRE_U32 {
    PERFORATOR_WIRE_ENUM(SAMPLE_TYPE_UNDEFINED, Undefined) = 0,
    PERFORATOR_WIRE_ENUM(SAMPLE_TYPE_PERF_EVENT, PerfEvent) = 1,
    PERFORATOR_WIRE_ENUM(SAMPLE_TYPE_KPROBE_FINISH_TASK_SWITCH, KprobeFinishTaskSwitch) = 2,
    PERFORATOR_WIRE_ENUM(SAMPLE_TYPE_TRACEPOINT_SIGNAL_DELIVER, TracepointSignalDeliver) = 3,
    PERFORATOR_WIRE_ENUM(SAMPLE_TYPE_TRACEPOINT_SCHED_SWITCH, TracepointSchedSwitch) = 4,
    PERFORATOR_WIRE_ENUM(SAMPLE_TYPE_UPROBE, Uprobe) = 5,
};

enum language_id: PERFORATOR_WIRE_U8 {
    PERFORATOR_WIRE_ENUM(LANGUAGE_PYTHON, Python) = PERFORATOR_LANGUAGE_PYTHON,
    PERFORATOR_WIRE_ENUM(LANGUAGE_PHP, Php) = PERFORATOR_LANGUAGE_PHP,
    PERFORATOR_WIRE_ENUM(LANGUAGE_JVM, Jvm) = PERFORATOR_LANGUAGE_JVM,
    PERFORATOR_WIRE_ENUM(LANGUAGE_LUA, Lua) = PERFORATOR_LANGUAGE_LUA,
};

enum language_payload_kind: PERFORATOR_WIRE_U8 {
    // Interpreter frames; the native parser keeps their payload opaque.
    PERFORATOR_WIRE_ENUM(LANGUAGE_PAYLOAD_INTERPRETER_FRAMES, InterpreterFrames) = 1,
    // Entries annotate native stack positions and therefore have to be
    // interpreted by the native stack builder (currently JVM only).
    PERFORATOR_WIRE_ENUM(LANGUAGE_PAYLOAD_NATIVE_ANNOTATIONS, NativeAnnotations) = 2,
};

enum tls_variable_type: PERFORATOR_WIRE_U8 {
    PERFORATOR_WIRE_ENUM(THREAD_LOCAL_UINT64_TYPE, Uint64) = 1,
    PERFORATOR_WIRE_ENUM(THREAD_LOCAL_STRING_TYPE, String) = 2,
};

enum {
    PACKED_SAMPLE_MAX_DATA = 8192,
    MAX_THREAD_LOCAL_STRING_LENGTH = 128,
};

enum cgroup_limits {
    PARENT_CGROUP_MAX_LEVELS = 16,
};

enum cgroup_consts: PERFORATOR_WIRE_U64 {
    END_OF_CGROUP_LIST = (PERFORATOR_WIRE_U64)-1,
};

struct branch_record {
    PERFORATOR_WIRE_U64 PERFORATOR_WIRE_FIELD(from, From);
    PERFORATOR_WIRE_U64 PERFORATOR_WIRE_FIELD(to, To);
    // Preserve the 24-byte perf_branch_entry layout used by bpf_read_branch_records.
    PERFORATOR_WIRE_U64 PERFORATOR_WIRE_FIELD(flags, Flags);
};

struct thread_local_string {
    PERFORATOR_WIRE_U64 PERFORATOR_WIRE_FIELD(len, Length);
    char PERFORATOR_WIRE_FIELD(string, Data)[MAX_THREAD_LOCAL_STRING_LENGTH];
};

union thread_local_variable {
    struct thread_local_string PERFORATOR_WIRE_FIELD(string, String);
    PERFORATOR_WIRE_U64 PERFORATOR_WIRE_FIELD(number, Number);
};

struct thread_local_variable_collect_result {
    PERFORATOR_WIRE_U64 PERFORATOR_WIRE_FIELD(offset, Offset);
    enum tls_variable_type PERFORATOR_WIRE_FIELD(type, Type);
    union thread_local_variable PERFORATOR_WIRE_FIELD(value, Value);
};

struct perf_event_attr_subset {
    PERFORATOR_WIRE_U32 PERFORATOR_WIRE_FIELD(type, Type);
    PERFORATOR_WIRE_U64 PERFORATOR_WIRE_FIELD(config, Config);
};

struct perf_event_subset {
    struct perf_event_attr_subset PERFORATOR_WIRE_FIELD(attr, Attr);
    PERFORATOR_WIRE_U64 PERFORATOR_WIRE_FIELD(id, Id);
};

union sample_config {
    PERFORATOR_WIRE_I32 PERFORATOR_WIRE_FIELD(sig, Signal);
    struct perf_event_subset PERFORATOR_WIRE_FIELD(perf_event, PerfEvent);
    PERFORATOR_WIRE_U64 PERFORATOR_WIRE_FIELD(ip, Ip);
};

struct record_new_process {
    enum record_tag PERFORATOR_WIRE_FIELD(tag, Tag);
    PERFORATOR_WIRE_U32 PERFORATOR_WIRE_FIELD(pid, Pid);
    PERFORATOR_WIRE_U64 PERFORATOR_WIRE_FIELD(starttime, StartTime);
};

struct language_section_header {
    // Payload size in bytes, excluding this header.
    PERFORATOR_WIRE_U16 PERFORATOR_WIRE_FIELD(byte_size, ByteSize);
    enum language_id PERFORATOR_WIRE_FIELD(language, Language);
    enum language_payload_kind PERFORATOR_WIRE_FIELD(payload_kind, PayloadKind);
    PERFORATOR_WIRE_U16 PERFORATOR_WIRE_FIELD(element_size, ElementSize);
    PERFORATOR_WIRE_U8 PERFORATOR_WIRE_FIELD(_pad, Padding)[2];
};

struct jvm_lang_entry {
    PERFORATOR_WIRE_U16 PERFORATOR_WIRE_FIELD(user_stack_index, UserStackIndex);
    PERFORATOR_WIRE_U8 PERFORATOR_WIRE_FIELD(_pad, Padding)[6];
    PERFORATOR_WIRE_U64 PERFORATOR_WIRE_FIELD(method_addr, MethodAddress);
};

struct section_desc {
    // Byte offset from the start of packed_sample.data.
    PERFORATOR_WIRE_U16 PERFORATOR_WIRE_FIELD(offset, Offset);
    PERFORATOR_WIRE_U16 PERFORATOR_WIRE_FIELD(size, Size);
};

struct record_sample_header {
    enum record_tag PERFORATOR_WIRE_FIELD(tag, Tag);
    enum sample_type PERFORATOR_WIRE_FIELD(sample_type, SampleType);
    union sample_config PERFORATOR_WIRE_FIELD(sample_config, SampleConfig);
    bool PERFORATOR_WIRE_FIELD(kthread, Kthread);
    PERFORATOR_WIRE_U16 PERFORATOR_WIRE_FIELD(cpu, Cpu);
    PERFORATOR_WIRE_U32 PERFORATOR_WIRE_FIELD(runtime, Runtime);
    PERFORATOR_WIRE_U64 PERFORATOR_WIRE_FIELD(collection_time, CollectionTime);
    PERFORATOR_WIRE_U8 PERFORATOR_WIRE_FIELD(thread_comm, ThreadComm)[16];
    PERFORATOR_WIRE_U8 PERFORATOR_WIRE_FIELD(process_comm, ProcessComm)[16];
    PERFORATOR_WIRE_U32 PERFORATOR_WIRE_FIELD(pid, Pid);
    PERFORATOR_WIRE_U32 PERFORATOR_WIRE_FIELD(tid, Tid);
    PERFORATOR_WIRE_U32 PERFORATOR_WIRE_FIELD(innermost_pidns_tid, InnermostPidnsTid);
    PERFORATOR_WIRE_U32 PERFORATOR_WIRE_FIELD(innermost_pidns_pid, InnermostPidnsPid);
    PERFORATOR_WIRE_U64 PERFORATOR_WIRE_FIELD(parent_cgroup, ParentCgroup);
    PERFORATOR_WIRE_U64 PERFORATOR_WIRE_FIELD(starttime, StartTime);
    PERFORATOR_WIRE_U64 PERFORATOR_WIRE_FIELD(value, Value);
    PERFORATOR_WIRE_U64 PERFORATOR_WIRE_FIELD(timedelta, TimeDelta);
    struct section_desc PERFORATOR_WIRE_FIELD(kern_stack, KernStack);
    struct section_desc PERFORATOR_WIRE_FIELD(user_stack, UserStack);
    struct section_desc PERFORATOR_WIRE_FIELD(lbr, Lbr);
    struct section_desc PERFORATOR_WIRE_FIELD(tls, Tls);
    struct section_desc PERFORATOR_WIRE_FIELD(cgroups, Cgroups);
    struct section_desc PERFORATOR_WIRE_FIELD(language_sections, LanguageSections);
};

#ifdef PERFORATOR_WIRE_DEFAULT_MACROS
    #undef PERFORATOR_WIRE_U8
    #undef PERFORATOR_WIRE_U16
    #undef PERFORATOR_WIRE_U32
    #undef PERFORATOR_WIRE_U64
    #undef PERFORATOR_WIRE_I32
    #undef PERFORATOR_WIRE_FIELD
    #undef PERFORATOR_WIRE_ENUM
    #undef PERFORATOR_WIRE_DEFAULT_MACROS
#endif
