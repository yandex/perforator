#pragma once

#include "api.h"
#include "../dwarf.h"
#include "../unwind_ctx.h"

static ALWAYS_INLINE long jvm_read(void* dst, u8 size, u64 src) {
    long res = bpf_probe_read_user(dst, size, (const void*) src);
    if (res < 0) {
        BPF_TRACE("[jvm] bpf_probe_read_user failed (%u bytes at 0x%lx): %lu", size, src, -res);
    }
    return res;
}

static ALWAYS_INLINE struct jvm_binary_config* jvm_get_bin_config(struct mapped_binary* libjvm) {
    binary_id id = libjvm->id;
    return bpf_map_lookup_elem(&jvm_binaries, &id);
}

static ALWAYS_INLINE struct jvm_process_config* jvm_get_proc_config(u32 pid) {
    return bpf_map_lookup_elem(&jvm_processes, &pid);
}

static ALWAYS_INLINE int jvm_process_interpreted_frame(
    struct jvm_binary_config* jc,
    struct unwind_context* regs,
    struct stack* stack,
    u64* method_ptr
) {
    u64 method_ptr_addr = regs->fp + jc->interpreter_stack_frame_method_offset * 8;
    BPF_TRACE("[jvm] method_ptr located at %llX\n", method_ptr_addr);
    long ret = jvm_read(method_ptr, sizeof(method_ptr), method_ptr_addr);
    if (ret < 0) {
        BPF_TRACE("[jvm] failed to read JVM method pointer: %lld\n", -ret);
        return -1;
    }
    BPF_TRACE("[jvm] method_ptr: %llX\n", *method_ptr);

    u64 caller_ip_addr = regs->fp + jc->stack_frame_return_addr_offset * 8;
    BPF_TRACE("[jvm] return address located af %llX\n", caller_ip_addr);
    u64 caller_ip = 0;
    ret = jvm_read(&caller_ip, sizeof(caller_ip), caller_ip_addr);
    if (ret < 0) {
        BPF_TRACE("[jvm] failed to read return address: %lld\n", -ret);
        return -1;
    }
    BPF_TRACE("[jvm] return address: %llX\n", caller_ip);
    regs->ip = caller_ip;
    ret = jvm_read(&regs->cfa, sizeof(regs->cfa), regs->fp - 8);
    if (ret < 0) {
        BPF_TRACE("[jvm] failed to read CFA: %lld\n", -ret);
        return -1;
    }
    u64 caller_rbp = 0;
    BPF_TRACE("[jvm] caller_rbp located at %llX\n", regs->fp);
    ret = jvm_read(&caller_rbp, sizeof(caller_rbp), regs->fp);
    if (ret < 0) {
        BPF_TRACE("[jvm] failed to read caller rbp: %lld\n", -ret);
        return -1;
    }
    regs->fp = caller_rbp;
    BPF_TRACE("[jvm] frame done\n");
    return 0;
}

enum {
    JVM_UNWIND_STATUS_OK = 0,
    JVM_UNWIND_STATUS_OVERFLOW = 101,
    JVM_UNWIND_STATUS_ERROR = 102,
};

// Staging context for JVM frame collection (passed as single pointer to
// stay within BPF's 5-argument limit for function calls).
struct jvm_staging {
    struct jvm_lang_entry* entries;
    u8* count;
};

// Collect one JVM interpreted frame. Stores the real native IP in stack->ips[]
// and records (user_stack_index, method_addr) in the JVM staging buffer.
static NOINLINE int jvm_collect_stack_v2(
    struct unwind_context* regs,
    struct jvm_binary_config* jc,
    struct stack* stack,
    struct jvm_staging* staging
) {
    u64 method_addr = 0;

    int res = jvm_process_interpreted_frame(jc, regs, stack, &method_addr);
    if (res < 0) {
        BPF_TRACE("[jvm] failed to process frame, error=%d\n", -res);
        return -JVM_UNWIND_STATUS_ERROR;
    }

    u8 jvm_idx = *staging->count;
    BPF_TRACE("[jvm] processed frame, recording at index %d, jvm index %d\n", stack->len, jvm_idx);
    if (jvm_idx >= MAX_JVM_FRAMES) {
        BPF_TRACE("[jvm] too many JVM frames, stopping\n");
        return -JVM_UNWIND_STATUS_OVERFLOW;
    }
    if (stack->len >= STACK_SIZE) {
        BPF_TRACE("[jvm] too many stack frames, stopping\n");
        return -JVM_UNWIND_STATUS_OVERFLOW;
    }

    // Store JVM placeholder in user stack (actual method resolved via jvm_lang_entry)
    u32 idx = stack->len;
    stack->ips[idx] = 0xFFFFFFFFDEADF00D;
    stack->len++;

    // Store (index, method_addr) in JVM staging
    staging->entries[jvm_idx].user_stack_index = (u16)idx;
    staging->entries[jvm_idx].method_addr = method_addr;
    *staging->count = jvm_idx + 1;

    return JVM_UNWIND_STATUS_OK;
}
