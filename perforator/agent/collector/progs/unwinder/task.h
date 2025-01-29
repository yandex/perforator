#pragma once

#include "core.h"

struct task_struct* get_current_task() {
    return (void*)bpf_get_current_task();
}

ALWAYS_INLINE bool is_kthread() {
    struct task_struct* task = get_current_task();
    return BPF_CORE_READ(task, flags) & PF_KTHREAD;
}

ALWAYS_INLINE struct task_struct* get_task_process(struct task_struct* task) {
    return BPF_CORE_READ(task, group_leader);
}

ALWAYS_INLINE struct task_struct* get_current_process() {
    struct task_struct* task = get_current_task();
    return get_task_process(task);
}

ALWAYS_INLINE u64 get_current_process_start_time() {
    struct task_struct* process = get_current_process();

    if (BPF_CORE_FIELD_EXISTS(process->real_start_time)) {
        // 5.4
        return BPF_CORE_READ(process, real_start_time);
    } else {
        // 5.15
        struct task_struct___v15* new_task = (void*)process;
        return BPF_CORE_READ(new_task, start_boottime);
    }
}

ALWAYS_INLINE void get_current_process_comm(u8 buf[TASK_COMM_LEN]) {
    struct task_struct* process = get_current_process();

    bpf_core_read_str(buf, TASK_COMM_LEN, &process->comm);
}
