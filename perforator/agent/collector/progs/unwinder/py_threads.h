#pragma once

#include "metrics.h"
#include "pidns.h"
#include "pthread.h"
#include "py_types.h"
#include "thread.h"

#include <bpf/bpf.h>

#include <stddef.h>

static ALWAYS_INLINE void* python_read_py_thread_state_ptr_static_tls(u64 offset) {
    unsigned long tcb = get_tcb_pointer();

    BPF_TRACE("python: read tcb %p, offset %d", tcb, offset);

    void* uaddr = (void*) (tcb - offset);

    void* py_thread_state_addr = NULL;
    long err = bpf_probe_read_user(&py_thread_state_addr, sizeof(void*), uaddr);
    if (err != 0) {
        metric_increment(METRIC_PYTHON_READ_TLS_THREAD_STATE_ERROR_COUNT);
        BPF_TRACE("python: failed to read thread local *Pythread_state from user space memory %p: %d", uaddr, err);
        return NULL;
    }

    return py_thread_state_addr;
}

static ALWAYS_INLINE bool read_tss_key(u32* key_dst, struct python_state* state) {
    if (state == NULL || key_dst == NULL || state->auto_tss_key_address == 0) {
        return false;
    }

    u32 is_initialized = 0;
    long err = bpf_probe_read_user(&is_initialized, sizeof(u32), (void*) (state->auto_tss_key_address + state->config.offsets.py_tss_t_offsets.is_initialized));
    if (err != 0) {
        BPF_TRACE("python: failed to read is_initialized at address %p: %d", state->auto_tss_key_address + state->config.offsets.py_tss_t_offsets.is_initialized, err);
        return false;
    }

    if (is_initialized == 0) {
        BPF_TRACE("python: tss is not initialized, auto tss key address %p, offset %d", state->auto_tss_key_address, state->config.offsets.py_tss_t_offsets.is_initialized);
        return false;
    }

    err = bpf_probe_read_user(key_dst, sizeof(u32), (void*) (state->auto_tss_key_address + state->config.offsets.py_tss_t_offsets.key));
    if (err != 0) {
        BPF_TRACE("python: failed to read tss key at address %p: %d", state->auto_tss_key_address + state->config.offsets.py_tss_t_offsets.key, err);
        return false;
    }

    return true;
}

static ALWAYS_INLINE void* python_read_py_thread_state_ptr_pthread_tss(struct python_state* state) {
    if (state == NULL) {
        return NULL;
    }

    if (state->auto_tss_key_address == 0) {
        BPF_TRACE("python: no auto tss key address found");
        return NULL;
    }

    u32 tss_key = 0;
    if (!read_tss_key(&tss_key, state)) {
        BPF_TRACE("python: failed to read tss key");
        return NULL;
    }

    BPF_TRACE("python: read tss key %u", tss_key);

    return pthread_read_tss(&state->pthread_config, tss_key);
}

static ALWAYS_INLINE void* python_read_py_thread_state_ptr_from_tls(struct python_state* state) {
    if (state == NULL) {
        return NULL;
    }

    if (state->config.py_thread_state_tls_offset != 0) {
        return python_read_py_thread_state_ptr_static_tls(state->config.py_thread_state_tls_offset);
    } else if (state->found_pthread_config) {
        return python_read_py_thread_state_ptr_pthread_tss(state);
    } else {
        BPF_TRACE("python: no tls read tried");
    }

    return NULL;
}

static ALWAYS_INLINE void* python_get_py_thread_state_from_cache(u32 current_ns_pid, u32 inner_ns_tid) {
    struct python_thread_key key = {
        .current_ns_pid = current_ns_pid,
        .inner_ns_tid = inner_ns_tid
    };

    void** py_thread_state_ptr = bpf_map_lookup_elem(&python_thread_id_py_thread_state, &key);
    if (py_thread_state_ptr == NULL) {
        BPF_TRACE("python: failed to find PyThreadState for current_ns_pid=%u, inner_ns_tid=%u",
                 current_ns_pid, inner_ns_tid);
        return NULL;
    }

    BPF_TRACE("python: successfully retrieved PyThreadState for current_ns_pid=%u, inner_ns_tid=%u",
             current_ns_pid, inner_ns_tid);

    return *py_thread_state_ptr;
}

static ALWAYS_INLINE void* python_get_current_thread_state_from_cache(u32 current_ns_pid) {
    u32 inner_ns_tid = get_current_task_innermost_pid();
    return python_get_py_thread_state_from_cache(current_ns_pid, inner_ns_tid);
}

static ALWAYS_INLINE u32 python_read_native_thread_id(void* py_thread_state, struct python_thread_state_offsets* thread_state_offsets) {
    if (py_thread_state == NULL || thread_state_offsets == NULL || thread_state_offsets->native_thread_id == PYTHON_UNSPECIFIED_OFFSET) {
        return 0;
    }

    u32 native_thread_id = 0;
    long err = bpf_probe_read_user(&native_thread_id, sizeof(u32), (void*)py_thread_state + thread_state_offsets->native_thread_id);
    if (err != 0) {
        metric_increment(METRIC_PYTHON_READ_NATIVE_THREAD_ID_ERROR_COUNT);
        BPF_TRACE(
            "python: failed to read native thread ID at offset %d: %d",
            thread_state_offsets->native_thread_id,
            err
        );
        return 0;
    }

    return native_thread_id;
}

static NOINLINE void python_upsert_thread_state(struct python_state* state, void* py_thread_state) {
    if (state == NULL || py_thread_state == NULL) {
        return;
    }

    BPF_TRACE("python: upsert PyThreadState");

    state->thread_key.current_ns_pid = state->pid;

    // Here we assume that native_thread_id is actually pid in bottom-level pid namespace.
    state->thread_key.inner_ns_tid = python_read_native_thread_id(py_thread_state, &state->config.offsets.py_thread_state_offsets);
    if (state->thread_key.inner_ns_tid == 0) {
        BPF_TRACE("python: failed to retrieve native thread ID from thread_state %p", py_thread_state);
        return;
    }

    long err = bpf_map_update_elem(&python_thread_id_py_thread_state, &state->thread_key, &py_thread_state, BPF_ANY);
    if (err != 0) {
        BPF_TRACE("python: failed to update BPF map with native_thread_id=%u: %d",
                 state->thread_key.inner_ns_tid, err);
    }

    BPF_TRACE("python: successfully upserted PyThreadState %p for native_thread_id=%u",
             (void*) py_thread_state, state->thread_key.inner_ns_tid);
}

static ALWAYS_INLINE void* python_retrieve_main_interpreterstate(void* py_runtime_ptr, struct python_runtime_state_offsets* runtime_state_offsets) {
    if (py_runtime_ptr == NULL || runtime_state_offsets == NULL) {
        return NULL;
    }

    void* main_interpreter_state = NULL;
    long err = bpf_probe_read_user(
        &main_interpreter_state,
        sizeof(void*),
        py_runtime_ptr + runtime_state_offsets->py_interpreters_main
    );
    if (err != 0) {
        BPF_TRACE("python: failed to read main PyInterpreterState: %d", err);
        return NULL;
    }
    if (main_interpreter_state == NULL) {
        BPF_TRACE("python: main *PyInterpreterState is NULL");
        return NULL;
    }

    BPF_TRACE("python: successfully retrieved main PyInterpreterState %p", main_interpreter_state);

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
        py_interpreter_state + interpreter_state_offsets->threads_head
    );
    if (err != 0) {
        BPF_TRACE("python: failed to read head *PyThreadState from *PyInterpreterState: %d", err);
        return NULL;
    }

    return head_thread_state;
}

static ALWAYS_INLINE void* python_get_head_thread_state(
    void* py_runtime_ptr,
    struct python_internals_offsets* offsets
) {
    if (py_runtime_ptr == NULL || offsets == NULL) {
        return NULL;
    }

    void* main_interpreter_state = python_retrieve_main_interpreterstate(py_runtime_ptr, &offsets->py_runtime_state_offsets);
    void* head_thread_state = python_retrieve_thread_state_from_interpreterstate(main_interpreter_state, &offsets->py_interpreter_state_offsets);

    if (head_thread_state == NULL) {
        BPF_TRACE("python: head *PyThreadState from *PyInterpreterState is NULL");
    }

    BPF_TRACE("python: successfully retrieved head *PyThreadState from *PyInterpreterState");

    return head_thread_state;
}

static NOINLINE void* python_read_next_thread_state(void* py_thread_state, struct python_thread_state_offsets* thread_state_offsets) {
    if (py_thread_state == NULL || thread_state_offsets == NULL) {
        return NULL;
    }

    void* next_thread_state = NULL;
    long err = bpf_probe_read_user(&next_thread_state, sizeof(void*), (void*)py_thread_state + thread_state_offsets->next_thread);
    if (err != 0) {
        BPF_TRACE("python: failed to read next *PyThreadState: %d", err);
        return NULL;
    }

    return next_thread_state;
}

static NOINLINE void* python_read_prev_thread_state(void* py_thread_state, struct python_thread_state_offsets* thread_state_offsets) {
    if (py_thread_state == NULL || thread_state_offsets == NULL) {
        return NULL;
    }

    void* prev_thread_state = NULL;
    long err = bpf_probe_read_user(&prev_thread_state, sizeof(void*), (void*)py_thread_state + thread_state_offsets->prev_thread);
    if (err != 0) {
        BPF_TRACE("python: failed to read prev *PyThreadState: %d", err);
        return NULL;
    }

    return prev_thread_state;
}

static ALWAYS_INLINE void python_fill_threads_cache(struct python_state* state, void* py_thread_state) {
    if (py_thread_state == NULL || state == NULL) {
        return;
    }

    void* current_thread_state = py_thread_state;
    for (u32 i = 0; i < MAX_PYTHON_THREAD_STATE_WALK && current_thread_state != NULL; i++) {;
        python_upsert_thread_state(state, current_thread_state);
        current_thread_state = python_read_next_thread_state(current_thread_state, &state->config.offsets.py_thread_state_offsets);
    }

    current_thread_state = py_thread_state;
    for (u32 i = 0; i < MAX_PYTHON_THREAD_STATE_WALK && current_thread_state != NULL; i++) {
        python_upsert_thread_state(state, current_thread_state);
        current_thread_state = python_read_prev_thread_state(current_thread_state, &state->config.offsets.py_thread_state_offsets);
    }
}

static ALWAYS_INLINE void* python_get_thread_state_and_update_cache(
    struct python_state* state
) {
    if (state == NULL) {
        return NULL;
    }

    // Attempt to read the PyThreadState pointer from TLS
    void* current_thread_state = python_read_py_thread_state_ptr_from_tls(state);
    void* fill_cache_thread_state = current_thread_state;
    if (fill_cache_thread_state == NULL) {
        fill_cache_thread_state = python_get_head_thread_state((void*) state->py_runtime_address, &state->config.offsets);
    } else {
        BPF_TRACE("python: successfully retrieved PyThreadState from TLS: %p", fill_cache_thread_state);
    }

    python_fill_threads_cache(state, fill_cache_thread_state);

    if (current_thread_state == NULL) {
        current_thread_state = python_get_current_thread_state_from_cache(state->pid);
    }

    if (current_thread_state == NULL) {
        BPF_TRACE("python: failed to retrieve PyThreadState from both TLS and cache for thread");
    }

    return current_thread_state;
}
