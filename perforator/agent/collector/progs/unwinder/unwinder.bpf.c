#ifndef __KERNEL__
#define __KERNEL__
#endif

#ifdef __x86_64__

#include "arch/x86/regs.h"

#elif __aarch64__

#include "arch/arm/regs.h"

#else

#error This arch is not supported by Perforator yet

#endif

#include "cgroups.h"
#include "core.h"
#include "dwarf.h"
#include "lbr.h"
#include "metrics.h"
#include "output.h"
#include "pidns.h"
#include "python/unwind.h"
#include "php/unwind.h"
#include "lua/unwind.h"
#include "task.h"
#include "thread_local.h"
#include "tracepoints.h"

#include "jvm/api.h"
#include "jvm/unwind.h"

#include <bpf/bpf.h>

#include <linux/bpf.h>
#include <linux/perf_event.h>
#include <linux/signal.h>


////////////////////////////////////////////////////////////////////////////////

struct perf_event_value {
    u64 counter;
    u64 samples;
};

struct stack_key {
    u32 pid;
    u32 tid;
    u64 cgroup;
    u32 starttime;
    u32 userstack;
    u32 kernstack;
    char comm[TASK_COMM_LEN];
};

enum {
    PROCESS_MAP_SIZE = 65536,
    SAMPLES_MAP_SIZE = 1 << 20,
    STACK_MAPS_SIZE = 1 << 16,
};

////////////////////////////////////////////////////////////////////////////////

struct profiler_state {
    struct packed_sample packed;

    u64 iteration;
    u64 prog_starttime;
    bool normalize_walltime;
    bool record_walltime;
    u64 task_cgroups[PARENT_CGROUP_MAX_LEVELS];
    struct pt_regs regs;

    u64 traced_cgroup;

    bool skip_sample_recording;
    bool should_trace_process;

    u64 event_count;
    struct perf_event_value sum;
    struct bpf_perf_event_value prev_counter;

    struct stack kernstack;
    struct stack userstack;

    // JVM scratch: only needed for mixed unwinding (interleaved native/JVM frames).
    struct jvm_lang_entry jvm_entries[MAX_JVM_FRAMES];
    u8 jvm_frames_count;

    struct python_state python_state;
    struct php_state php_state;
    struct lua_state lua_state;

    struct last_branch_records lbr;
    struct record_new_process newproc;
    struct tls_collect_result tls;
};

BTF_EXPORT(enum { signal_mask_bits = 64 });

enum cgroup_engine {
    CGROUP_ENGINE_UNSPECIFIED,
    CGROUP_ENGINE_V1,
    CGROUP_ENGINE_V2
};

BTF_EXPORT(enum cgroup_engine);

struct profiler_config {
    // Include samples from the kthreads in the output.
    bool trace_kthreads;

    // Trace whole system, excluding kthreads.
    bool trace_whole_system;

    // Enable JVM-specific unwinding and symbolization.
    bool enable_jvm;

    // Enable PHP profiling
    bool enable_php;

    // Enable Lua profiling
    bool enable_lua;

    // Cgroup resolution engine to use
    enum cgroup_engine active_cgroup_engine;

    // Collect samples from this process only.
    // Set @pid_filter to zero to disable.
    int pid_filter;

    // Inode number of the pid namespace to resolve pids at.
    // Useful if your profiler is run inside pid namespace.
    // If not set, the program will return top-level pids.
    u32 pidns_inode;

    // Analyze only 1/sample_modulo sched events.
    u64 sched_sample_modulo;

    // Signal set to sample.
    // If signals SIGSMTH should be sampled, then (profiler_config->signal_mask & (1 << SIGSMTH)) != 0.
    u64 signal_mask;
    _Static_assert(SIGRTMIN < signal_mask_bits, "Unsupported signal set size");
};

struct thread_key {
    u32 pid;
    u32 tid;
    u64 starttime;
};

// Heap. BPF stack is limited (512 bytes).
BPF_MAP(profiler_state, BPF_MAP_TYPE_PERCPU_ARRAY, u32, struct profiler_state, 1);
BPF_MAP(profiler_config, BPF_MAP_TYPE_ARRAY, u32, struct profiler_config, 1);
BPF_MAP(process_info, BPF_MAP_TYPE_HASH, u32, struct process_info, PROCESS_MAP_SIZE);
BPF_MAP(default_process_info, BPF_MAP_TYPE_ARRAY, u32, struct process_info, 1);
BPF_MAP(process_discovery, BPF_MAP_TYPE_LRU_HASH, u32, u8, PROCESS_MAP_SIZE);
BPF_MAP(perf_event_values, BPF_MAP_TYPE_HASH, u64, struct bpf_perf_event_value, 4096);
BPF_MAP(thread_last_sample_time, BPF_MAP_TYPE_LRU_HASH, struct thread_key, u64, 256 * 1024);
BPF_MAP(percpu_user_regs, BPF_MAP_TYPE_PERCPU_ARRAY, u32, struct user_regs, 1);

static ALWAYS_INLINE void* map_lookup_zero(void* map) {
    u32 zero = 0;
    return bpf_map_lookup_elem(map, &zero);
}

static ALWAYS_INLINE void try_get_stack(void* ctx, struct stack* stack, u64 flags) {
    long res = bpf_get_stack(ctx, stack->ips, sizeof(stack->ips), flags);
    if (res < 0) {
        BPF_TRACE("bpf_get_stack(%llu) failed: %d\n", flags, res);
        stack->len = 0;
        return;
    }
    // bpf_get_stack returns bytes; normalize to frame count.
    stack->len = res / sizeof(u64);
}

// Copy `bytes` from src into packed->data at offset.
static NOINLINE u32 pack_copy(struct packed_sample* packed, u32 offset, void* src, u32 bytes) {
    if (bytes == 0) return 0;
    BPF_VALUE_BARRIER(bytes);
    BPF_VALUE_BARRIER(offset);
    bytes = bytes & (PACKED_SAMPLE_MAX_DATA - 1);
    offset = offset & (PACKED_SAMPLE_MAX_DATA - 1);
    if (offset + bytes > PACKED_SAMPLE_MAX_DATA) return 0;
    bpf_probe_read(packed->data + offset, bytes, src);
    return bytes;
}

static ALWAYS_INLINE void set_section(struct section_desc* desc, u32 offset, u32 size) {
    desc->offset = (u16)offset;
    desc->size = (u16)size;
}

// Re-narrow verifier range of `offset` to avoid state-space blowup.
#define CLAMP_OFFSET(off) do { \
    BPF_VALUE_BARRIER(off); \
    (off) &= (PACKED_SAMPLE_MAX_DATA - 1); \
} while (0)

// Pack kern/user stack, LBR, TLS, cgroups.
static NOINLINE u32 pack_sample_core(struct packed_sample* packed, struct profiler_state* state) {
    struct record_sample_header* hdr = &packed->header;
    u32 offset = 0;
    u32 written;

    // Zero descriptors: early return must produce a valid empty header.
    ZERO(hdr->kern_stack);
    ZERO(hdr->user_stack);
    ZERO(hdr->lbr);
    ZERO(hdr->tls);
    ZERO(hdr->cgroups);
    ZERO(hdr->language_sections);

    // Kernel stack
    u32 klen = state->kernstack.len;
    if (klen > PERF_MAX_STACK_DEPTH) klen = PERF_MAX_STACK_DEPTH;
    u32 kbytes = klen * sizeof(u64);
    if (kbytes > sizeof(state->kernstack.ips)) kbytes = sizeof(state->kernstack.ips);
    written = pack_copy(packed, offset, state->kernstack.ips, kbytes);
    set_section(&hdr->kern_stack, offset, written);
    offset += written;
    CLAMP_OFFSET(offset);

    // User stack
    u32 ulen = state->userstack.len;
    if (ulen > STACK_SIZE) ulen = STACK_SIZE;
    u32 ubytes = ulen * sizeof(u64);
    if (ubytes > sizeof(state->userstack.ips)) ubytes = sizeof(state->userstack.ips);
    written = pack_copy(packed, offset, state->userstack.ips, ubytes);
    set_section(&hdr->user_stack, offset, written);
    offset += written;
    CLAMP_OFFSET(offset);

    // LBR
    u32 lbr_nr = state->lbr.nr;
    if (lbr_nr > MAX_BRANCH_RECORDS) lbr_nr = MAX_BRANCH_RECORDS;
    u32 lbr_bytes = lbr_nr * sizeof(struct branch_record);
    if (lbr_bytes > sizeof(state->lbr.entries)) lbr_bytes = sizeof(state->lbr.entries);
    written = pack_copy(packed, offset, state->lbr.entries, lbr_bytes);
    set_section(&hdr->lbr, offset, written);
    offset += written;
    CLAMP_OFFSET(offset);

    // TLS: direct stores instead of pack_copy to keep the verifier budget low.
    u32 tls_start = offset;
    for (int i = 0; i < MAX_TRACKED_THREAD_LOCALS_PER_BINARY; i++) {
        if (state->tls.values[i].offset == 0) break;
        if (offset + sizeof(struct thread_local_variable_collect_result) > PACKED_SAMPLE_MAX_DATA) break;
        u32 tls_off = offset;
        CLAMP_OFFSET(tls_off);
        *(struct thread_local_variable_collect_result*)(packed->data + tls_off) = state->tls.values[i];
        offset += sizeof(struct thread_local_variable_collect_result);
        CLAMP_OFFSET(offset);
    }
    set_section(&hdr->tls, tls_start, offset - tls_start);

    // Cgroups: see TLS note above.
    u32 cg_start = offset;
    for (int i = 0; i < PARENT_CGROUP_MAX_LEVELS; i++) {
        u64 cg = state->task_cgroups[i];
        if (cg == END_OF_CGROUP_LIST) break;
        if (offset + sizeof(u64) > PACKED_SAMPLE_MAX_DATA) break;
        u32 cg_off = offset;
        CLAMP_OFFSET(cg_off);
        *(u64*)(packed->data + cg_off) = cg;
        offset += sizeof(u64);
        CLAMP_OFFSET(offset);
    }
    set_section(&hdr->cgroups, cg_start, offset - cg_start);

    return offset;
}

// Descriptor for a single language section to be packed.
struct lang_section_desc {
    u8 lang_id;
    u32 count;
    u32 max;
    void* src;
    u32 src_size;
    u32 elem_size;
};

// Pack one language section into the packed sample buffer.
// Takes section parameters via a descriptor struct to stay within BPF's
// 5-argument limit for subprograms. CLAMP_OFFSET keeps verifier range narrow.
static ALWAYS_INLINE void pack_lang_section(
    struct packed_sample* packed,
    u32* offset,
    struct lang_section_desc* desc
) {
    if (desc->count == 0) return;
    u32 fc = desc->count;
    if (fc > desc->max) fc = desc->max;
    u32 bytes = fc * desc->elem_size;
    if (bytes > desc->src_size) bytes = desc->src_size;
    BPF_VALUE_BARRIER(bytes);
    bytes &= (PACKED_SAMPLE_MAX_DATA - 1);
    u32 off = *offset;
    CLAMP_OFFSET(off);
    if (off + sizeof(struct language_section_header) + bytes <= PACKED_SAMPLE_MAX_DATA) {
        struct language_section_header* lsh =
            (struct language_section_header*)(packed->data + off);
        lsh->byte_size = bytes;
        lsh->language = desc->lang_id;
        __builtin_memset(lsh->_pad, 0, sizeof(lsh->_pad));
        off += sizeof(struct language_section_header);
        CLAMP_OFFSET(off);
        pack_copy(packed, off, desc->src, bytes);
        off += bytes;
        CLAMP_OFFSET(off);
        *offset = off;
    }
}

static NOINLINE u32 pack_sample_lang(struct packed_sample* packed, struct profiler_state* state, u32 offset) {
    u32 lang_start = offset;

    pack_lang_section(packed, &offset, &(struct lang_section_desc){
        .lang_id   = LANGUAGE_PYTHON,
        .count     = state->python_state.frame_count,
        .max       = PYTHON_MAX_STACK_DEPTH,
        .src       = state->python_state.frames,
        .src_size  = sizeof(state->python_state.frames),
        .elem_size = sizeof(struct interpreter_frame),
    });

    pack_lang_section(packed, &offset, &(struct lang_section_desc){
        .lang_id   = LANGUAGE_PHP,
        .count     = state->php_state.frame_count,
        .max       = PHP_MAX_STACK_DEPTH,
        .src       = state->php_state.frames,
        .src_size  = sizeof(state->php_state.frames),
        .elem_size = sizeof(struct interpreter_frame),
    });

    pack_lang_section(packed, &offset, &(struct lang_section_desc){
        .lang_id   = LANGUAGE_JVM,
        .count     = state->jvm_frames_count,
        .max       = MAX_JVM_FRAMES,
        .src       = state->jvm_entries,
        .src_size  = sizeof(state->jvm_entries),
        .elem_size = sizeof(struct jvm_lang_entry),
    });

    pack_lang_section(packed, &offset, &(struct lang_section_desc){
        .lang_id   = LANGUAGE_LUA,
        .count     = state->lua_state.stack.len,
        .max       = LUA_MAX_STACK_DEPTH,
        .src       = state->lua_state.stack.frames,
        .src_size  = sizeof(state->lua_state.stack.frames),
        .elem_size = sizeof(struct interpreter_frame),
    });

    set_section(&packed->header.language_sections, lang_start, offset - lang_start);
    return offset;
}

static ALWAYS_INLINE u32 pack_sample(struct packed_sample* packed, struct profiler_state* state) {
    u32 offset = pack_sample_core(packed, state);
    offset = pack_sample_lang(packed, state, offset);
    return sizeof(struct record_sample_header) + offset;
}

static ALWAYS_INLINE void record_sample(
    void* ctx,
    struct profiler_state* state,
    u64 value
) {
    struct record_sample_header* hdr = &state->packed.header;

    hdr->parent_cgroup = state->traced_cgroup;
    hdr->value = value;
    hdr->cpu = bpf_get_smp_processor_id();

    u64 ktime = bpf_ktime_get_ns();
    hdr->runtime = ktime - state->prog_starttime;
    hdr->collection_time = state->prog_starttime;

    u32 size = pack_sample(&state->packed, state);
    submit_packed_sample(ctx, &state->packed, size);

    if (value) {
        __sync_fetch_and_add(&state->sum.counter, value);
    }
    __sync_fetch_and_add(&state->sum.samples, 1);
}

static NOINLINE struct process_info* lookup_process(
    void* ctx,
    struct profiler_state* state
) {
    u32 pid = state->packed.header.pid;
    u64 starttime = state->packed.header.starttime;

    struct process_info* info = bpf_map_lookup_elem(&process_info, &pid);
    if (info != 0) {
        return info;
    }
    metric_increment(METRIC_PROCESS_UNKNOWN_COUNT);

    // Use default unwind info.
    // We cannot unwind stack using DWARF without process_info,
    // so we try to fallback to frame pointers.
    info = map_lookup_zero(&default_process_info);
    if (!info) {
        return NULL;
    }
    info->unwind_type = UNWIND_TYPE_FP;

    // Notify userspace about the new process.
    u8 value = 0;
    if (bpf_map_update_elem(&process_discovery, &pid, &value, BPF_NOEXIST)) {
        return 0;
    }

    state->newproc.pid = pid;
    state->newproc.starttime = starttime;
    submit_new_process(ctx, &state->newproc);
    metric_increment(METRIC_PROCESS_NOTIFIED_COUNT);
    return 0;
}

static ALWAYS_INLINE u64 get_perf_event_id(struct bpf_perf_event_data* ctx) {
    struct bpf_perf_event_data_kern* kctx = (void*)ctx;
    return BPF_CORE_READ(kctx, event, id);
}

static ALWAYS_INLINE u32 get_perf_event_type(struct bpf_perf_event_data* ctx) {
    struct bpf_perf_event_data_kern* kctx = (void*)ctx;
    return BPF_CORE_READ(kctx, event, attr.type);
}

static ALWAYS_INLINE u64 get_perf_event_config(struct bpf_perf_event_data* ctx) {
    struct bpf_perf_event_data_kern* kctx = (void*)ctx;
    return BPF_CORE_READ(kctx, event, attr.config);
}

static NOINLINE u64 calculate_perf_counter_delta(struct bpf_perf_event_data* ctx, u64 id) {
    struct bpf_perf_event_value zero = {};
    struct bpf_perf_event_value* prev = bpf_map_lookup_elem(&perf_event_values, &id);
    if (!prev) {
        prev = &zero;
    }

    struct bpf_perf_event_value value;
    int err = bpf_perf_prog_read_value(ctx, &value, sizeof(value));
    if (err != 0) {
        BPF_TRACE("bpf_perf_prog_read_value failed: %d\n", err);
        return 0;
    }

    if (prev->counter > value.counter) {
        prev->counter = 0;
        prev->running = 0;
        prev->enabled = 0;
    }

    struct bpf_perf_event_value delta = {
        .counter = value.counter - prev->counter,
        .enabled = value.enabled - prev->enabled,
        .running = value.running - prev->running,
    };

    prev = &value;

    err = bpf_map_update_elem(&perf_event_values, &id, prev, BPF_ANY);
    if (err != 0) {
        BPF_TRACE("bpf_map_update_elem failed: %d", err);
        return 0;
    }

    if (delta.counter == 0 || delta.enabled == 0 || delta.running == 0) {
        BPF_TRACE("zero event: %lld, %lld, %lld\n", delta.counter, delta.enabled, delta.running);
        return 0;
    }

    u64 ratio = delta.running * 100 / delta.enabled;
    u64 count = delta.counter * delta.enabled / delta.running;
    if (ratio != 100) {
        metric_add(METRIC_PERFEVENT_MULTIPLEXED_COUNT, 1);
        BPF_TRACE("unexpected ratio: %lld, scaling %lld -> %lld\n", ratio, delta.counter, count);
        BPF_TRACE("dcounter: %lld, drunning: %lld, denabled: %lld\n", delta.counter, delta.running, delta.enabled);
    }

    return count;
}

static ALWAYS_INLINE struct profiler_state* get_state() {
    struct profiler_state* state = map_lookup_zero(&profiler_state);
    if (state == 0) {
        BPF_TRACE("failed to get profiler state\n");
    }
    return state;
}

static ALWAYS_INLINE struct profiler_config* get_config() {
    struct profiler_config* config = map_lookup_zero(&profiler_config);

    if (config == 0) {
        BPF_TRACE("failed to get profiler config\n");
    }
    return config;
}

////////////////////////////////////////////////////////////////////////////////

static ALWAYS_INLINE void record_thread_walltime(struct profiler_config* config, struct profiler_state* state) {
    if (!state->record_walltime) {
        return;
    }

    struct record_sample_header* hdr = &state->packed.header;
    struct thread_key key = {
        .pid = hdr->pid,
        .tid = hdr->tid,
        .starttime = hdr->starttime,
    };

    u64* last = bpf_map_lookup_elem(&thread_last_sample_time, &key);
    if (last) {
        i64 delta = (i64)state->prog_starttime - (i64)*last;
        BPF_TRACE("calculated thread %d timedelta: %lld ns\n", hdr->tid, delta);
        if (delta >= 0) {
            hdr->timedelta = (u64)delta;
        }
    } else {
        BPF_TRACE("found thread without previous sample time\n", hdr->tid);
        hdr->timedelta = 0;
    }

    if (state->normalize_walltime) {
        hdr->timedelta *= config->sched_sample_modulo;
    }

    int err = bpf_map_update_elem(&thread_last_sample_time, &key, &state->prog_starttime, 0);
    if (err != 0) {
        BPF_TRACE("failed to set thread %d sample time: %d\n", hdr->tid, err);
    }
}

static NOINLINE int profiler_stage_start(void* ctx, struct profiler_state* state, struct profiler_config* config) {
    if (state == NULL || config == NULL) {
        return -101;
    }

    state->iteration++;

    // Skip kernel threads.
    bool kthread = is_kthread();
    if (!config->trace_kthreads && kthread) {
        metric_increment(METRIC_FILTERED_KTHREAD_COUNT);
        return -102;
    }

    struct record_sample_header* hdr = &state->packed.header;

    // Collect some basic info about current thread.
    u64 pid_tgid = get_current_pidns_pid_tgid(config->pidns_inode);
    hdr->tid = (u32)pid_tgid;
    hdr->pid = pid_tgid >> 32;
    struct task_struct* task = get_current_task();
    hdr->innermost_pidns_tid = get_task_innermost_pidns_pid(task);
    hdr->innermost_pidns_pid = get_task_innermost_pidns_pid(get_task_process(task));
    hdr->starttime = get_current_process_start_time();
    hdr->kthread = kthread;

    if (config->pid_filter != 0 && config->pid_filter != hdr->pid) {
        metric_increment(METRIC_FILTERED_PROCESS_COUNT);
        return -103;
    }

    record_thread_walltime(config, state);

    if (state->skip_sample_recording) {
        return -104;
    }

    return 0;
}

static ALWAYS_INLINE bool should_trace_process(u32 tgid, u32 pid) {
    if (bpf_map_lookup_elem(&traced_processes, &tgid) != NULL) {
        return true;
    }
    if (bpf_map_lookup_elem(&traced_processes, &pid) != NULL) {
        return true;
    }
    return false;
}

static NOINLINE int profiler_stage_locate_traceee(struct profiler_state* state, struct profiler_config* config) {
    if (state == NULL || config == NULL) {
        return -201;
    }

    switch (config->active_cgroup_engine) {
        case CGROUP_ENGINE_V1: {
            get_current_cgroup_hierarchy_v1(state->task_cgroups, &state->traced_cgroup);
            break;
        }
        case CGROUP_ENGINE_V2: {
            get_current_cgroup_hierarchy_v2(state->task_cgroups, &state->traced_cgroup);
            break;
        }
        default: {
            BPF_TRACE("Invalid config: unknown cgroup engine requested: %d\n", config->active_cgroup_engine);
            return -203;
        }
    }

    if (config->trace_whole_system) {
        return 0;
    }

    state->should_trace_process = should_trace_process(state->packed.header.pid, state->packed.header.tid);

    if (state->traced_cgroup == END_OF_CGROUP_LIST && !state->should_trace_process) {
        // Non-traced cgroup & task. Skip it.
        return -202;
    }

    return 0;
}

static ALWAYS_INLINE int mixed_collect_stack(struct user_regs* regs, struct stack* stack, struct process_info* proc, struct profiler_state* state) {
    u32 pid = state->packed.header.pid;
#ifndef PERFORATOR_COMPAT_5_4
    struct jvm_binary_config* jvm_bin_config = NULL;
    struct jvm_process_config* jvm_proc_config = NULL;
    if (is_mapped(proc->libjvm_binary)) {
        jvm_bin_config = jvm_get_bin_config(&proc->libjvm_binary);
        if (jvm_bin_config != NULL) {
            jvm_proc_config = jvm_get_proc_config(pid);
        }
    }
#endif

    struct dwarf_unwind_context* ctx = dwarf_get_context();
    if (ctx == NULL) {
        DWARF_TRACE("[mixed_unwinder] failed to load DWARF unwinder state from heap\n");
        return 0;
    }
    if (!dwarf_unwind_init(ctx, regs, pid)) {
        DWARF_TRACE("[mixed_unwinder] failed to retrieve userspace registers\n");
        return 0;
    }

    int res = -1;
    for (int i = 0; i < DWARF_UNWIND_MAX_STACK_SIZE; ++i) {
#ifndef PERFORATOR_COMPAT_5_4
        bool is_jvm = false;
#endif
        u64 rip = ctx->cfi.ip;
#ifdef PERFORATOR_COMPAT_5_4
        (void) rip;
#endif

        BPF_TRACE("[mixed_unwinder] start iteration %d: processing address %llx\n", i, ctx->cfi.ip);
#ifndef PERFORATOR_COMPAT_5_4
        if (jvm_proc_config != NULL) {
            if (jvm_proc_config->interpreter_begin <= rip && rip < jvm_proc_config->interpreter_end) {
                BPF_TRACE("[jvm] address 0x%llx is in JVM interpreter\n", rip);
                is_jvm = true;
            }
        }
        if (is_jvm) {
            BPF_TRACE("[mixed_unwinder] using JVM unwinder for this frame\n");
            struct jvm_staging staging = {
                .entries = state->jvm_entries,
                .count = &state->jvm_frames_count,
            };
            int jvm_res = jvm_collect_stack_v2(
                &ctx->cfi, jvm_bin_config, stack, &staging
            );
            if (jvm_res < 0) {
                BPF_TRACE("[jvm] failed to collect stack, error=%d\n", -jvm_res);
                res = -1;
                goto done;
            }
        } else
#endif
        {
            BPF_TRACE("[mixed_unwinder] using DWARF unwinder for this frame\n");
            if (!dwarf_unwind_record_stack_frame(ctx, stack)) {
                DWARF_TRACE("[mixed_unwinder] failed to record DWARF stack frame\n");
                goto done;
            }

            enum dwarf_unwind_step_result step_result = dwarf_unwind_step();
            switch (step_result) {
            case DWARF_UNWIND_STEP_FAILED:
                DWARF_TRACE("[mixed_unwinder] DWARF unwinding failed at step %d: %d\n", i, ctx->error);
                res = -1;
                goto done;
            case DWARF_UNWIND_STEP_FINISHED:
                DWARF_TRACE("[mixed_unwinder] DWARF unwinding finished\n");
                res = 0;
                goto done;
            case DWARF_UNWIND_STEP_CONTINUE:
                break;
            }
        }
    }

    BPF_TRACE("[mixed_unwinder] stack limit exhausted\n");

done:
    metric_add(METRIC_STACK_FRAME_DWARF_COUNT, stack->len - ctx->framepointers);
    metric_add(METRIC_STACK_FRAME_FP_COUNT, ctx->framepointers);
    metric_add(METRIC_STACK_FRAME_COUNT, stack->len);
    BPF_TRACE("[mixed_unwinder] found %u/%u frames using frame pointers\n", ctx->framepointers, stack->len);
    return res;
}

static ALWAYS_INLINE int profiler_stage_collect_stack(void* ctx, struct user_regs* regs, struct profiler_state* state, struct profiler_config* config) {
    if (state == NULL || config == NULL) {
        return -301;
    }

    struct process_info* proc = lookup_process(ctx, state);
    if (!proc) {
        BPF_TRACE("Unknown process %d\n", state->packed.header.pid);
        return -302;
    }

    ZERO(state->kernstack);
    try_get_stack(ctx, &state->kernstack, 0);

    ZERO(state->userstack);
    state->jvm_frames_count = 0;
    switch (proc->unwind_type) {
    case UNWIND_TYPE_FP:
        try_get_stack(ctx, &state->userstack, BPF_F_USER_STACK);
        break;
    case UNWIND_TYPE_DISABLED:
        break;
    case UNWIND_TYPE_MIXED:
        mixed_collect_stack(regs, &state->userstack, proc, state);
        break;
    default:
        BPF_TRACE("Unsupported unwind type %d\n", proc->unwind_type);
        return 0;
    }

    return 0;
}

static NOINLINE int profiler_stage_collect_python_stack(void* ctx, struct profiler_state* state, struct profiler_config* config) {
    if (state == NULL || config == NULL) {
        return -1;
    }

    struct process_info* info = lookup_process(ctx, state);
    if (!info) {
        return -1;
    }

    state->python_state.pid = state->packed.header.pid;
    state->python_state.frame_count = 0;
    state->python_state.py_runtime_address = 0;
    python_collect_stack(info, &state->python_state);
    return 0;
}

static NOINLINE int profiler_stage_collect_php_stack(void* ctx, struct profiler_state* state, struct profiler_config* config) {
    if (state == NULL || config == NULL) {
        return -1;
    }
    if (!config->enable_php) {
        return 0;
    }

    struct process_info* info = lookup_process(ctx, state);
    if (!info) {
        return -1;
    }
    php_collect_stack(info, &state->php_state);
    return 0;
}

/**
 * @brief LuaJIT VM stack unwinder entrypoint.
 *
 * @see `lua/unwind.h`
 *
 * @param context BPF context.
 * @param user_registers Struct of user registers.
 * @param profiler_state Profiler state.
 * @return int Status code
 */
static NOINLINE int profiler_stage_collect_lua_stack(void* context, const struct user_regs* user_registers, struct profiler_state* profiler_state, struct profiler_config* config) {
#ifdef __x86_64__
    if (!config->enable_lua) {
        return 0;
    }

    struct process_info* process_info = lookup_process(context, profiler_state);
    if (!process_info) {
        return -1;
    }

    struct lua_state* lua_state = &profiler_state->lua_state;
    lua_state->pid = profiler_state->packed.header.pid;
    lua_state_init_registers(lua_state, user_registers);

    lua_collect_stack(process_info, lua_state);

    return 0;
#elif __aarch64__
    // Not implemented yet
    return 0;
#endif
}

static NOINLINE int profiler_stage_collect_tls(void* ctx, struct profiler_state* state, struct profiler_config* config) {
    if (state == NULL || config == NULL) {
        return -1;
    }

    struct process_info* info = lookup_process(ctx, state);
    if (!info) {
        return -1;
    }

    collect_tls_values(info, &state->tls);
    return 0;
}

static NOINLINE int profiler_stage_record_sample(
    void* ctx,
    struct profiler_state* state
) {
    if (state == NULL || ctx == NULL) {
        return -1;
    }

    struct record_sample_header* hdr = &state->packed.header;
    ZERO(hdr->process_comm);
    get_current_process_comm(hdr->process_comm);
    bpf_get_current_comm(hdr->thread_comm, ARRAY_SIZE(hdr->thread_comm));

    record_sample(ctx, state, state->event_count);
    return 0;
}

static NOINLINE int profiler_stage_collect_lbr_stack(void* ctx, struct profiler_state* state) {
    if (ctx == NULL || state == NULL) {
        return -1;
    }

    collect_lbr_stack(ctx, &state->lbr);
    return 0;
}

////////////////////////////////////////////////////////////////////////////////

struct profiler_sample_args {
    u64 event_count;
    u64 starttime;
    enum sample_type sample_type;
    union sample_config sample_config;
    bool needs_lbr_stack;
    bool normalize_walltime;
    bool record_walltime;
    bool skip_sample_recording;
};

#define PROFILER_DO_SAMPLE_COMMON_PROLOGUE \
    struct profiler_state* state = get_state(); \
    if (!state) { \
        return -1; \
    } \
 \
    struct profiler_config* config = get_config(); \
    if (!config) { \
        return -2; \
    } \
 \
    state->prog_starttime = args->starttime; \
    state->event_count = args->event_count; \
    state->packed.header.sample_type = args->sample_type; \
    state->packed.header.sample_config = args->sample_config; \
    state->normalize_walltime = args->normalize_walltime; \
    state->record_walltime = args->record_walltime; \
    state->skip_sample_recording = args->skip_sample_recording; \
    state->jvm_frames_count = 0; \
    state->lbr.nr = 0; \
    state->userstack.len = 0; \
\
    int err = 0; \

#define PROFILER_DO_SAMPLE_COMMON_EPILOGUE \
    if ((err = profiler_stage_record_sample(ctx, state)) != 0) { \
        metric_increment(METRIC_ERROR_STAGE_RECORDSAMPLE_COUNT); \
        return err; \
    } \
 \
    return 0; \

#define PROFILER_DEFINE_STAGE(fn, metric) \
    if ((err = fn) != 0) { \
        metric_increment(metric); \
        return err; \
    } \

#define PROFILER_DEFINE_COMMON_STAGES \
    PROFILER_DEFINE_STAGE(profiler_stage_start(ctx, state, config), METRIC_ERROR_STAGE_START_COUNT); \
    PROFILER_DEFINE_STAGE(profiler_stage_locate_traceee(state, config), METRIC_ERROR_STAGE_LOCATETRACEEE_COUNT); \
    PROFILER_DEFINE_STAGE(profiler_stage_collect_stack(ctx, regs, state, config), METRIC_ERROR_STAGE_COLLECTSTACK_COUNT); \
    PROFILER_DEFINE_STAGE(profiler_stage_collect_tls(ctx, state, config), METRIC_ERROR_STAGE_TLS_COUNT); \
    PROFILER_DEFINE_STAGE(profiler_stage_collect_python_stack(ctx, state, config), METRIC_ERROR_STAGE_COLLECT_PYTHON_STACK_COUNT); \
    PROFILER_DEFINE_STAGE(profiler_stage_collect_lua_stack(ctx, regs, state, config), METRIC_ERROR_STAGE_COLLECT_LUA_STACK_COUNT); \

static NOINLINE int profiler_do_sample_impl_perfevent(void* ctx, struct user_regs* regs, struct profiler_sample_args* args) {
    PROFILER_DO_SAMPLE_COMMON_PROLOGUE;

    PROFILER_DEFINE_COMMON_STAGES;
    PROFILER_DEFINE_STAGE(profiler_stage_collect_php_stack(ctx, state, config), METRIC_ERROR_STAGE_COLLECT_PHP_STACK_COUNT)
    PROFILER_DEFINE_STAGE(profiler_stage_collect_lbr_stack(ctx, state), METRIC_ERROR_STAGE_LBR_STACK_COUNT);

    PROFILER_DO_SAMPLE_COMMON_EPILOGUE;
}

static NOINLINE int profiler_do_sample_impl_other(void* ctx, struct user_regs* regs, struct profiler_sample_args* args) {
    PROFILER_DO_SAMPLE_COMMON_PROLOGUE;

    PROFILER_DEFINE_COMMON_STAGES;

    PROFILER_DO_SAMPLE_COMMON_EPILOGUE;
}

// We split profiler_do_sample into three NOINLINE functions:
// - profiler_do_sample_start
// - profiler_do_sample_impl
// - profiler_do_sample_finish
// In order to reduce stack usage.
static NOINLINE void profiler_do_sample_start(struct profiler_sample_args* args) {
    metric_add(METRIC_EVENT_COUNT, args->event_count);
    metric_increment(METRIC_SAMPLE_COUNT);
}

static NOINLINE void profiler_do_sample_finish(int err) {
    if (err != 0) {
        BPF_TRACE("unwinder failed with error code: %d\n", err);
        metric_increment(METRIC_SAMPLE_UNSUCCESSFULL_COUNT);
    } else {
        metric_increment(METRIC_SAMPLE_SUCCESSFULL_COUNT);
    }
}

static NOINLINE void profiler_do_sample_perfevent(void* ctx, struct user_regs* regs, struct profiler_sample_args* args) {
    profiler_do_sample_start(args);
    int err = profiler_do_sample_impl_perfevent(ctx, regs, args);
    profiler_do_sample_finish(err);
}

static NOINLINE void profiler_do_sample_other(void* ctx, struct user_regs* regs, struct profiler_sample_args* args) {
    profiler_do_sample_start(args);
    int err = profiler_do_sample_impl_other(ctx, regs, args);
    profiler_do_sample_finish(err);
}

////////////////////////////////////////////////////////////////////////////////

SEC("perf_event")
int perforator_perf_event(struct bpf_perf_event_data* ctx) {
    struct profiler_sample_args args = {};
    args.starttime = bpf_ktime_get_ns();
    args.needs_lbr_stack = true;
    args.record_walltime = true;

    args.sample_type = SAMPLE_TYPE_PERF_EVENT;
    args.sample_config.perf_event.attr.type = get_perf_event_type(ctx);
    args.sample_config.perf_event.attr.config = get_perf_event_config(ctx);
    args.sample_config.perf_event.id = get_perf_event_id(ctx);
    args.event_count = calculate_perf_counter_delta(ctx, args.sample_config.perf_event.id);
    if (args.event_count == 0) {
        return 0;
    }
    BPF_TRACE("Got event count %llu\n", args.event_count);

    struct user_regs* regs = map_lookup_zero(&percpu_user_regs);
    if (regs == NULL) {
        return 0;
    }
    if (!find_task_userspace_registers_bpf(&ctx->regs, regs)) {
        BPF_TRACE("Failed to load perf user_regs\n");
        return 0;
    }

    profiler_do_sample_perfevent(ctx, regs, &args);

    return 0;
}

////////////////////////////////////////////////////////////////////////////////

static NOINLINE int profiler_do_sample_amd_fam19h_brs_perfevent(void* ctx, struct profiler_sample_args* args) {
    PROFILER_DO_SAMPLE_COMMON_PROLOGUE;

    PROFILER_DEFINE_STAGE(profiler_stage_start(ctx, state, config), METRIC_ERROR_STAGE_START_COUNT);
    PROFILER_DEFINE_STAGE(profiler_stage_locate_traceee(state, config), METRIC_ERROR_STAGE_LOCATETRACEEE_COUNT);
    PROFILER_DEFINE_STAGE(profiler_stage_collect_tls(ctx, state, config), METRIC_ERROR_STAGE_TLS_COUNT);
    PROFILER_DEFINE_STAGE(profiler_stage_collect_lbr_stack(ctx, state), METRIC_ERROR_STAGE_LBR_STACK_COUNT);

    PROFILER_DO_SAMPLE_COMMON_EPILOGUE;
}

SEC("perf_event")
int perforator_amd_fam19h_brs_event(struct bpf_perf_event_data* ctx) {
    struct profiler_sample_args args = {};
    args.starttime = bpf_ktime_get_ns();
    args.needs_lbr_stack = true;
    args.record_walltime = false;

    args.sample_type = SAMPLE_TYPE_PERF_EVENT;
    args.sample_config.perf_event.attr.type = get_perf_event_type(ctx);
    args.sample_config.perf_event.attr.config = get_perf_event_config(ctx);
    args.sample_config.perf_event.id = get_perf_event_id(ctx);
    args.event_count = calculate_perf_counter_delta(ctx, args.sample_config.perf_event.id);
    if (args.event_count == 0) {
        return 0;
    }
    BPF_TRACE("Got event count %llu\n", args.event_count);

    profiler_do_sample_start(&args);
    int err = profiler_do_sample_amd_fam19h_brs_perfevent(ctx, &args);
    profiler_do_sample_finish(err);

    return 0;
}

////////////////////////////////////////////////////////////////////////////////

static NOINLINE bool sample_sched_event() {
    struct profiler_config* config = get_config();
    if (!config) {
        return false;
    }

    if (config->sched_sample_modulo == 0) {
        BPF_TRACE("sample_sched_event: no modulo\n");
        return false;
    }

    u32 rng = bpf_get_prandom_u32();
    BPF_TRACE("sample_sched_event: %d %% %d\n", rng, config->sched_sample_modulo);

    return rng % config->sched_sample_modulo == 0;
}

SEC("kprobe/finish_task_switch")
int perforator_finish_task_switch(struct pt_regs* ctx) {
    struct profiler_sample_args args = {};
    args.starttime = bpf_ktime_get_ns();
    args.sample_type = SAMPLE_TYPE_KPROBE_FINISH_TASK_SWITCH;
    args.event_count = 0;
    args.needs_lbr_stack = false;
    args.normalize_walltime = true;
    args.record_walltime = true;
    args.skip_sample_recording = !sample_sched_event();

    struct user_regs* regs = map_lookup_zero(&percpu_user_regs);
    if (regs == NULL) {
        return 0;
    }

    if (!extract_saved_userspace_registers(regs)) {
        BPF_TRACE("Failed to load kprobe user_regs\n");
        return 1;
    }

    profiler_do_sample_other(ctx, regs, &args);

    return 0;
}

////////////////////////////////////////////////////////////////////////////////

static NOINLINE bool sample_signal(struct trace_event_signal_deliver* signal) {
    if (signal->sa_handler == (u64)SIG_IGN) {
        return false;
    }

    struct profiler_config* config = get_config();
    if (config == NULL) {
        return false;
    }

    u64 sigbit = (u64)1 << signal->sig;
    return config->signal_mask & sigbit;
}

SEC("tracepoint/signal_deliver")
int perforator_signal_deliver(struct trace_event_signal_deliver* ctx) {
    metric_increment(METRIC_SIGNALDELIVER_TRIGGERED_COUNT);
    if (!sample_signal(ctx)) {
        return 0;
    }
    metric_increment(METRIC_SIGNALDELIVER_SAMPLED_COUNT);

    struct profiler_sample_args args = {};
    args.starttime = bpf_ktime_get_ns();
    args.sample_type = SAMPLE_TYPE_TRACEPOINT_SIGNAL_DELIVER;
    args.sample_config.sig = ctx->sig;
    args.event_count = 0;
    args.needs_lbr_stack = false;
    args.record_walltime = false;

    struct user_regs* regs = map_lookup_zero(&percpu_user_regs);
    if (regs == NULL) {
        return 0;
    }

    if (!extract_saved_userspace_registers(regs)) {
        BPF_TRACE("Failed to load user_regs\n");
        return 1;
    }

    profiler_do_sample_other(ctx, regs, &args);

    return 0;
}

////////////////////////////////////////////////////////////////////////////////

SEC("uprobe")
int perforator_uprobe(struct pt_regs* ctx) {
    metric_increment(METRIC_UPROBE_TRIGGERED_COUNT);

    struct profiler_sample_args args = {};
    args.starttime = bpf_ktime_get_ns();
    args.sample_type = SAMPLE_TYPE_UPROBE;
    args.event_count = 1;
    args.needs_lbr_stack = false;
    args.record_walltime = false;

    struct user_regs* regs = map_lookup_zero(&percpu_user_regs);
    if (regs == NULL) {
        return 0;
    }
    if (!find_task_userspace_registers(ctx, regs)) {
        BPF_TRACE("Failed to load user_regs\n");
        return 1;
    }
    args.sample_config.ip = regs_get_current_instruction(regs);

    profiler_do_sample_other(ctx, regs, &args);

    return 0;
}

////////////////////////////////////////////////////////////////////////////////

LICENSE("GPL");

////////////////////////////////////////////////////////////////////////////////
