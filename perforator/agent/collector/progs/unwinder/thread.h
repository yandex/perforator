#pragma once

#include "core.h"

// Returns pointer to TCB (Thread Control Block) structure
static ALWAYS_INLINE unsigned long get_tcb_pointer() {
    struct task_struct* task = (void*)bpf_get_current_task();
    return BPF_CORE_READ(task, thread.fsbase);
}
