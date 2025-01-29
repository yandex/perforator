#pragma once

#include "metrics.h"

#include <bpf/bpf.h>

#include <stddef.h>

enum {
    MAX_PYTHON_THREADS = 16384,
    MAX_PYTHON_THREAD_STATE_WALK = 32,
};

struct python_interpreter_state_offsets {
    u32 next_offset;
    u32 threads_head_offset;
};

struct python_runtime_state_offsets {
    u32 py_interpreters_main_offset;
};

struct python_thread_state_offsets {
    u32 cframe_offset;
    u32 current_frame_offset;
    u32 native_thread_id_offset;
    u32 prev_thread_offset;
    u32 next_thread_offset;
};

BPF_MAP(python_thread_id_py_thread_state, BPF_MAP_TYPE_LRU_HASH, u32, void*, MAX_PYTHON_THREADS);

static ALWAYS_INLINE void* python_read_py_thread_state_ptr_from_tls(u64 offset) {
    struct task_struct* task = (void*)bpf_get_current_task();

    unsigned long fsbase = BPF_CORE_READ(task, thread.fsbase);

    BPF_TRACE("python: read fsbase %p, offset %d", fsbase, offset);

    void* uaddr = (void*) (fsbase - offset);

    void* py_thread_state_addr = NULL;
    long err = bpf_probe_read_user(&py_thread_state_addr, sizeof(void*), uaddr);
    if (err != 0) {
        metric_increment(METRIC_PYTHON_READ_TLS_THREAD_STATE_ERROR_COUNT);
        BPF_TRACE("python: failed to read thread local *Pythread_state from user space memory %p: %d", uaddr, err);
        return NULL;
    }

    return py_thread_state_addr;
}

static ALWAYS_INLINE void* python_get_py_thread_state_from_cache(u32 native_thread_id) {
    void** py_thread_state_ptr = bpf_map_lookup_elem(&python_thread_id_py_thread_state, &native_thread_id);
    if (py_thread_state_ptr == NULL) {
        BPF_TRACE("python: failed to find Pythread_state for native thread ID %u", native_thread_id);
        return NULL;
    }

    BPF_TRACE("python: successfully retrieved Pythread_state for native thread ID %u", native_thread_id);
    return *py_thread_state_ptr;
}

static ALWAYS_INLINE void* python_get_current_thread_state_from_cache() {
    u32 task_pid = bpf_get_current_pid_tgid() & 0xFFFFFFFF;
    return python_get_py_thread_state_from_cache(task_pid);
}

static ALWAYS_INLINE u32 python_read_native_thread_id(void* py_thread_state, struct python_thread_state_offsets* thread_state_offsets) {
    if (py_thread_state == NULL || thread_state_offsets == NULL) {
        return 0;
    }

    u32 native_thread_id = 0;
    long err = bpf_probe_read_user(&native_thread_id, sizeof(u32), (void*)py_thread_state + thread_state_offsets->native_thread_id_offset);
    if (err != 0) {
        metric_increment(METRIC_PYTHON_READ_NATIVE_THREAD_ID_ERROR_COUNT);
        BPF_TRACE(
            "python: failed to read native thread ID at offset %d: %d",
            thread_state_offsets->native_thread_id_offset,
            err
        );
        return 0;
    }

    return native_thread_id;
}

static NOINLINE void python_upsert_thread_state(void* py_thread_state, struct python_thread_state_offsets* thread_state_offsets) {
    if (py_thread_state == NULL || thread_state_offsets == NULL) {
        return;
    }

    u32 native_thread_id = python_read_native_thread_id(py_thread_state, thread_state_offsets);
    if (native_thread_id == 0) {
        BPF_TRACE("python: failed to retrieve native thread ID from thread_state %p", py_thread_state);
        return;
    }

    long err = bpf_map_update_elem(&python_thread_id_py_thread_state, &native_thread_id, &py_thread_state, BPF_ANY);
    if (err != 0) {
        BPF_TRACE("python: failed to update BPF map with native thread ID %u: %d", native_thread_id, err);
    }
}

// Bypass ASLR
static ALWAYS_INLINE void* python_get_global_runtime_address(u64 py_runtime_relative_address) {
    struct task_struct* task = (void*)bpf_get_current_task();
    unsigned long base_addr = BPF_CORE_READ(task, mm, start_code);
    return (void*) (base_addr + py_runtime_relative_address);
}

static ALWAYS_INLINE void* python_retrieve_main_interpreterstate(u64 py_runtime_relative_address, struct python_runtime_state_offsets* runtime_state_offsets) {
    if (py_runtime_relative_address == 0 || runtime_state_offsets == NULL) {
        return NULL;
    }

    void* py_runtime_address = python_get_global_runtime_address(py_runtime_relative_address);

    void* main_interpreter_state = NULL;
    long err = bpf_probe_read_user(
        &main_interpreter_state,
        sizeof(void*),
        py_runtime_address + runtime_state_offsets->py_interpreters_main_offset
    );
    if (err != 0) {
        BPF_TRACE("python: failed to read main PyInterpreterState: %d", err);
        return NULL;
    }

    if (main_interpreter_state == NULL) {
        BPF_TRACE("python: main *PyInterpreterState is NULL");
        return NULL;
    }

    BPF_TRACE("python: successfully retrieved main *PyInterpreterState");
    return main_interpreter_state;
}

static ALWAYS_INLINE void* python_retrieve_thread_state_from_interpreterstate(void* py_interpreter_state, struct python_interpreter_state_offsets* interpreter_state_offsets) {
    if (py_interpreter_state == NULL || interpreter_state_offsets == NULL) {
        return NULL;
    }

    void* head_thread_state = NULL;
    long err = bpf_probe_read_user(
        &head_thread_state,
        sizeof(void*),
        py_interpreter_state + interpreter_state_offsets->threads_head_offset
    );
    if (err != 0) {
        BPF_TRACE("python: failed to read head *Pythread_state from *PyInterpreterState: %d", err);
        return NULL;
    }

    BPF_TRACE("python: successfully retrieved head *Pythread_state from *PyInterpreterState");
    return head_thread_state;
}

static ALWAYS_INLINE void* python_get_head_thread_state(
    u64 py_runtime_relative_address,
    struct python_runtime_state_offsets* runtime_state_offsets,
    struct python_interpreter_state_offsets* interpreter_state_offsets
) {
    if (py_runtime_relative_address == 0 || runtime_state_offsets == NULL || interpreter_state_offsets == NULL) {
        return NULL;
    }

    void* main_interpreter_state = python_retrieve_main_interpreterstate(py_runtime_relative_address, runtime_state_offsets);
    void* head_thread_state = python_retrieve_thread_state_from_interpreterstate(main_interpreter_state, interpreter_state_offsets);

    if (head_thread_state == NULL) {
        BPF_TRACE("python: head *Pythread_state from *PyInterpreterState is NULL");
    }

    return head_thread_state;
}

static NOINLINE void* python_read_next_thread_state(void* py_thread_state, struct python_thread_state_offsets* thread_state_offsets) {
    if (py_thread_state == NULL || thread_state_offsets == NULL) {
        return NULL;
    }

    void* next_thread_state = NULL;
    long err = bpf_probe_read_user(&next_thread_state, sizeof(void*), (void*)py_thread_state + thread_state_offsets->next_thread_offset);
    if (err != 0) {
        BPF_TRACE("python: failed to read next *Pythread_state: %d", err);
        return NULL;
    }

    return next_thread_state;
}

static ALWAYS_INLINE void* python_read_prev_thread_state(void* py_thread_state, struct python_thread_state_offsets* thread_state_offsets) {
    if (py_thread_state == NULL || thread_state_offsets == NULL) {
        return NULL;
    }

    void* prev_thread_state = NULL;
    long err = bpf_probe_read_user(&prev_thread_state, sizeof(void*), (void*)py_thread_state + thread_state_offsets->prev_thread_offset);
    if (err != 0) {
        BPF_TRACE("python: failed to read prev *Pythread_state: %d", err);
        return NULL;
    }

    return prev_thread_state;
}

static ALWAYS_INLINE void python_fill_threads_cache(void* py_thread_state, struct python_thread_state_offsets* thread_state_offsets) {
    if (py_thread_state == NULL || thread_state_offsets == NULL) {
        return;
    }

    void *forward_thread_state = py_thread_state;
    for (u32 i = 0; i < MAX_PYTHON_THREAD_STATE_WALK && forward_thread_state != NULL; i++) {
        python_upsert_thread_state(forward_thread_state, thread_state_offsets);
        forward_thread_state = python_read_next_thread_state(forward_thread_state, thread_state_offsets);
    }

    void *backward_thread_state = py_thread_state;
    for (u32 i = 0; i < MAX_PYTHON_THREAD_STATE_WALK && backward_thread_state != NULL; i++) {
        python_upsert_thread_state(backward_thread_state, thread_state_offsets);
        backward_thread_state = python_read_prev_thread_state(backward_thread_state, thread_state_offsets);
    }
}

static ALWAYS_INLINE void* python_get_thread_state_and_update_cache(
    u64 py_thread_state_tls_offset,
    u64 py_runtime_relative_address,
    struct python_runtime_state_offsets* runtime_state_offsets,
    struct python_interpreter_state_offsets* interpreter_state_offsets,
    struct python_thread_state_offsets* thread_state_offsets
) {
    if (runtime_state_offsets == NULL || interpreter_state_offsets == NULL || thread_state_offsets == NULL) {
        return NULL;
    }

    // Attempt to read the Pythread_state pointer from TLS
    void* current_thread_state = python_read_py_thread_state_ptr_from_tls(py_thread_state_tls_offset);

    void *fill_cache_thread_state = current_thread_state;
    if (fill_cache_thread_state == NULL) {
        fill_cache_thread_state = python_get_head_thread_state(py_runtime_relative_address, runtime_state_offsets, interpreter_state_offsets);
    }

    python_fill_threads_cache(fill_cache_thread_state, thread_state_offsets);

    if (current_thread_state == NULL) {
        current_thread_state = python_get_current_thread_state_from_cache();
    }

    if (current_thread_state == NULL) {
        BPF_TRACE("python: failed to retrieve Pythread_state from both TLS and cache for thread");
    }

    return current_thread_state;
}
