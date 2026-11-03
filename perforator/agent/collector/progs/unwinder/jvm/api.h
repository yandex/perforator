#pragma once

#include <bpf/bpf.h>

struct jvm_binary_config {
    i32 stack_frame_return_addr_offset;
    i32 interpreter_stack_frame_method_offset;

};

struct jvm_process_config {
    u64 interpreter_begin;
    u64 interpreter_end;
};

enum {
    MAX_JVM_BINARIES = 100,
    MAX_JVM_PROCESSES = 1000,
    MAX_JVM_FRAMES = 64,
};

BPF_MAP(jvm_binaries, BPF_MAP_TYPE_HASH, binary_id, struct jvm_binary_config, MAX_JVM_BINARIES);

BPF_MAP(jvm_processes, BPF_MAP_TYPE_HASH, u32, struct jvm_process_config, MAX_JVM_PROCESSES);

struct jvm_frame {
    u16 index;
    u8 _pad[6];
    u64 method_addr;
};

struct jvm_stack {
    struct jvm_frame frames[MAX_JVM_FRAMES];
    u32 frames_len;
};
