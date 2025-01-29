#pragma once

#include <bpf/types.h>

#include "core.h"
#include "task.h"

enum { PIDNS_LOOKUP_MAX_DEPTH = 32 };

static NOINLINE u32 get_task_pidns_pid(struct task_struct* task, u32 pidns_ino) {
    struct pid* pid = BPF_CORE_READ(task, thread_pid);
    int level = BPF_CORE_READ(pid, level);

    for (int i = 0; i < PIDNS_LOOKUP_MAX_DEPTH; i++) {
        if (i > level) {
            break;
        }

        struct upid upid = BPF_CORE_READ(pid, numbers[i]);
        u32 ino = BPF_CORE_READ(upid.ns, ns.inum);
        if (ino == pidns_ino) {
            return upid.nr;
        }
    }

    // Fallback to the top-level pid.
    return BPF_CORE_READ(pid, numbers[0].nr);
}

static NOINLINE u64 get_current_pidns_pid_tgid(u32 pidns_ino) {
    if (pidns_ino == 0) {
        return bpf_get_current_pid_tgid();
    }

    struct task_struct* task = get_current_task();
    u32 pid = get_task_pidns_pid(task, pidns_ino);

    struct task_struct* process = get_task_process(task);
    u32 tgid = get_task_pidns_pid(process, pidns_ino);

    return (((u64)tgid) << 32) | pid;
}
