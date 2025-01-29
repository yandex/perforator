#pragma once

#include "attrs.h"
#include "types.h"

#include <linux/bpf.h>
#include <linux/bpf_perf_event.h>


#define BPF_FUNC(NAME, ...) \
    (*bpf_##NAME)(__VA_ARGS__) MAYBE_UNUSED = (void*)BPF_FUNC_##NAME

struct task_struct;

static void* BPF_FUNC(map_lookup_elem, void* map, const void* key);
static int BPF_FUNC(map_update_elem, void* map, const void* key, const void* value, u64 flags);
static int BPF_FUNC(map_delete_elem, void* map, const void* key);
static int BPF_FUNC(trace_printk, const char* fmt, u32 fmt_size, ...);
static u64 BPF_FUNC(ktime_get_ns);
static u64 BPF_FUNC(get_prandom_u32);
static u64 BPF_FUNC(get_current_task, void);
static struct task_struct* BPF_FUNC(get_current_task_btf, void);
static u64 BPF_FUNC(get_current_pid_tgid, void);
static int BPF_FUNC(perf_event_read_value, void* map, u64 flags, struct bpf_perf_event_value* buf, u32 buf_size);
static int BPF_FUNC(perf_prog_read_value, struct bpf_perf_event_data* ctx, struct bpf_perf_event_value* buf, u32 buf_size);
static long BPF_FUNC(get_stack, void* ctx, void* buf, u32 size, u64 flags);
static int BPF_FUNC(get_stackid, void* ctx, void* map, u64 flags);
static int BPF_FUNC(get_current_comm, void* buf, u32 size_of_buf);
static int BPF_FUNC(probe_read, void* dst, u32 size, const void* unsafe_ptr);
static int BPF_FUNC(probe_read_kernel, void* dst, u32 size, const void* unsafe_ptr);
static int BPF_FUNC(probe_read_user, void* dst, u32 size, const void* unsafe_ptr);
static long BPF_FUNC(probe_read_str, void* dst, u32 size, const void* unsafe_ptr);
static long BPF_FUNC(probe_read_kernel_str, void* dst, u32 size, const void* unsafe_ptr);
static long BPF_FUNC(probe_read_user_str, void* dst, u32 size, const void* unsafe_ptr);
static int BPF_FUNC(tail_call, void* ctx, void* progs, u32 index);
static long BPF_FUNC(perf_event_output, void* ctx, void* map, u64 flags, void* data, u64 size);
static u32 BPF_FUNC(get_smp_processor_id);
static int BPF_FUNC(read_branch_records, void* ctx, void* buf, u32 size, u64 flags);
static void* BPF_FUNC(task_pt_regs, struct task_struct* task);

// LLVM's builtin
void* memcpy(void* dest, const void* src, unsigned long size);

#undef BPF_FUNC
