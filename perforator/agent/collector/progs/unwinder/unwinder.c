#ifndef __KERNEL__
#define __KERNEL__
#endif

#include "arch/x86/regs.h"

#include "cgroups.h"
#include "core.h"
#include "dwarf.h"
#include "lbr.h"
#include "metrics.h"
#include "output.h"
#include "pidns.h"
#include "python.h"
#include "task.h"
#include "thread_local.h"
#include "tracepoints.h"

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
    u64 iteration;
    u64 prog_starttime;
    bool normalize_walltime;
    bool record_walltime;
    u64 task_cgroups[PARENT_CGROUP_MAX_LEVELS];
    struct pt_regs regs;

    u64 traced_cgroup;
    u32 traced_process;

    bool skip_sample_recording;

    u64 event_count;
    struct perf_event_value sum;
    struct bpf_perf_event_value prev_counter;

    struct stack kernstack;
    struct stack userstack;
    struct python_state python_state;

    struct record_sample sample;
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

// Heap. BPF stack is limited (512 bytes).
BPF_MAP(profiler_state, BPF_MAP_TYPE_PERCPU_ARRAY, u32, struct profiler_state, 1);
BPF_MAP(profiler_config, BPF_MAP_TYPE_ARRAY, u32, struct profiler_config, 1);
BPF_MAP(process_info, BPF_MAP_TYPE_HASH, u32, struct process_info, PROCESS_MAP_SIZE);
BPF_MAP(default_process_info, BPF_MAP_TYPE_ARRAY, u32, struct process_info, 1);
BPF_MAP(process_discovery, BPF_MAP_TYPE_LRU_HASH, u32, u8, PROCESS_MAP_SIZE);
BPF_MAP(perf_event_values, BPF_MAP_TYPE_HASH, u64, struct bpf_perf_event_value, 4096);
BPF_MAP(thread_last_sample_time, BPF_MAP_TYPE_LRU_HASH, u32, u64, 1024 * 1024);
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
    stack->len = res;
}

static NOINLINE void fill_stack(struct stack* stack, u64* target) {
    BPF_TRACE("Filling stack of size %d\n", (int)stack->len);
    int i = 0;
    for (i = 0; i < STACK_SIZE; ++i) {
        if (i < stack->len) {
            target[i] = stack->ips[i];
        } else {
            target[i] = 0;
        }
    }
}

static NOINLINE void fill_python_stack(struct python_state* state, struct record_sample* target) {
    target->python_stack_len = state->frame_count;
    if (state->frame_count > 0) {
        for (int i = 0; i < PYTHON_MAX_STACK_DEPTH && i < state->frame_count; i++) {
            target->python_stack[i] = state->frames[i];
        }
    }
}

static ALWAYS_INLINE void record_sample(
    void* ctx,
    struct profiler_state* state,
    u64 value
) {
    struct record_sample* sample = &state->sample;
    sample->parent_cgroup = state->traced_cgroup;
    _Static_assert(sizeof(sample->cgroups_hierarchy) == sizeof(state->task_cgroups), "Array length mismatch");
    memcpy(sample->cgroups_hierarchy, state->task_cgroups, sizeof(state->task_cgroups));
    fill_stack(&state->userstack, sample->userstack);
    fill_stack(&state->kernstack, sample->kernstack);
    sample->value = value;
    sample->cpu = bpf_get_smp_processor_id();

    fill_python_stack(&state->python_state, sample);

    u64 ktime = bpf_ktime_get_ns();
    sample->runtime = ktime - state->prog_starttime;

    sample->tls_values = state->tls;

    submit_sample(ctx, sample);

    if (value) {
        __sync_fetch_and_add(&state->sum.counter, value);
    }
    __sync_fetch_and_add(&state->sum.samples, 1);
}

static NOINLINE struct process_info* lookup_process(
    void* ctx,
    struct profiler_state* state
) {
    u32 pid = state->sample.pid;
    u64 starttime = state->sample.starttime;

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
    struct profiler_config* state = map_lookup_zero(&profiler_config);
    if (state == 0) {
        BPF_TRACE("failed to get profiler config\n");
    }
    return state;
}

////////////////////////////////////////////////////////////////////////////////

static ALWAYS_INLINE void record_thread_walltime(struct profiler_config* config, struct profiler_state* state) {
    if (!state->record_walltime) {
        return;
    }

    u64* last = bpf_map_lookup_elem(&thread_last_sample_time, &state->sample.tid);
    if (last) {
        i64 delta = (i64)state->prog_starttime - (i64)*last;
        BPF_TRACE("calculated thread %d timedelta: %lld ns\n", state->sample.tid, delta);
        if (delta >= 0) {
            state->sample.timedelta = (u64)delta;
        }
    } else {
        BPF_TRACE("found thread without previous sample time\n", state->sample.tid);
        state->sample.timedelta = 0;
    }

    if (state->normalize_walltime) {
        state->sample.timedelta *= config->sched_sample_modulo;
    }

    int err = bpf_map_update_elem(&thread_last_sample_time, &state->sample.tid, &state->prog_starttime, 0);
    if (err != 0) {
        BPF_TRACE("failed to set thread %d sample time: %d\n", state->sample.tid, err);
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

    // Collect some basic info about current thread.
    u64 pid_tgid = get_current_pidns_pid_tgid(config->pidns_inode);
    state->sample.tid = (u32)pid_tgid;
    state->sample.pid = pid_tgid >> 32;
    state->sample.starttime = get_current_process_start_time();
    state->sample.kthread = kthread;

    if (config->pid_filter != 0 && config->pid_filter != state->sample.pid) {
        metric_increment(METRIC_FILTERED_PROCESS_COUNT);
        return -103;
    }

    record_thread_walltime(config, state);

    if (state->skip_sample_recording) {
        return -104;
    }

    return 0;
}

static ALWAYS_INLINE u32 get_current_traced_process(u32 pid) {
    if (bpf_map_lookup_elem(&traced_processes, &pid) != NULL) {
        return pid;
    }
    return -1;
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
    state->traced_process = get_current_traced_process(state->sample.pid);

    if (config->trace_whole_system) {
        return 0;
    }

    if (state->traced_cgroup == END_OF_CGROUP_LIST && state->traced_process == -1) {
        // Unknown process & cgroup. Skip it.
        return -202;
    }

    return 0;
}

static NOINLINE int profiler_stage_collect_stack(void* ctx, struct user_regs* regs, struct profiler_state* state, struct profiler_config* config) {
    if (state == NULL || config == NULL) {
        return -301;
    }

    struct process_info* proc = lookup_process(ctx, state);
    if (!proc) {
        BPF_TRACE("Unknown process %d\n", state->sample.pid);
        return -302;
    }

    try_get_stack(ctx, &state->kernstack, 0);

    ZERO(state->userstack);
    switch (proc->unwind_type) {
    case UNWIND_TYPE_FP:
        try_get_stack(ctx, &state->userstack, BPF_F_USER_STACK);
        break;
    case UNWIND_TYPE_DWARF:
        dwarf_collect_stack(regs, &state->userstack);
        break;
    case UNWIND_TYPE_DISABLED:
        break;
    default:
        BPF_TRACE("Unsupported unwind type %d", proc->unwind_type);
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

    state->python_state.pid = state->sample.pid;
    state->python_state.frame_count = 0;
    python_collect_stack(info, &state->python_state);
    return 0;
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

    ZERO(state->sample.process_comm);
    get_current_process_comm(state->sample.process_comm);
    bpf_get_current_comm(state->sample.thread_comm, ARRAY_SIZE(state->sample.thread_comm));

    record_sample(ctx, state, state->event_count);
    return 0;
}

static NOINLINE int profiler_stage_collect_lbr_stack(void* ctx, struct profiler_state* state) {
    if (ctx == NULL || state == NULL) {
        return -1;
    }

    collect_lbr_stack(ctx, &state->sample.lbr_values);
    return 0;
}

////////////////////////////////////////////////////////////////////////////////

struct profiler_sample_args {
    u64 event_count;
    u64 starttime;
    enum sample_type sample_type;
    u64 sample_config;
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
    state->sample.sample_type = args->sample_type; \
    state->sample.sample_config = args->sample_config; \
    state->normalize_walltime = args->normalize_walltime; \
    state->record_walltime = args->record_walltime; \
    state->skip_sample_recording = args->skip_sample_recording; \
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

static NOINLINE int profiler_do_sample_impl_perfevent(void* ctx, struct user_regs* regs, struct profiler_sample_args* args) {
    PROFILER_DO_SAMPLE_COMMON_PROLOGUE;

    PROFILER_DEFINE_COMMON_STAGES;
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
    args.sample_config = get_perf_event_id(ctx);
    args.event_count = calculate_perf_counter_delta(ctx, args.sample_config);
    if (args.event_count == 0) {
        return 0;
    }
    BPF_TRACE("Got event count %llu\n", args.event_count);

    struct user_regs* regs = map_lookup_zero(&percpu_user_regs);
    if (regs == NULL) {
        return 0;
    }
    if (!find_task_userspace_registers(&ctx->regs, regs)) {
        BPF_TRACE("Failed to load perf user_regs\n");
        return 0;
    }

    profiler_do_sample_perfevent(ctx, regs, &args);

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
    args.sample_config = 0;
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
    args.sample_config = ctx->sig;
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

LICENSE("GPL");

////////////////////////////////////////////////////////////////////////////////
