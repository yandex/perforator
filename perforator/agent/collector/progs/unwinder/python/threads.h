#pragma once

#include "../metrics.h"
#include "../pidns.h"
#include "../pthread.h"
#include "types.h"
#include "../thread.h"

#include <bpf/bpf.h>

#include <stddef.h>

static ALWAYS_INLINE void* python_read_py_thread_state_ptr_static_tls(u64 offset) {
    unsigned long tcb = get_tcb_pointer();

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

static ALWAYS_INLINE bool read_tss_key(i32* key_dst, struct python_state* state) {
    if (state == NULL || key_dst == NULL || state->auto_tss_key_address == 0) {
        return false;
    }

    long err;
    if (state->config.offsets.py_tss_t_offsets.is_initialized != PYTHON_UNSPECIFIED_OFFSET) {
        u32 is_initialized = 0;
        err = bpf_probe_read_user(&is_initialized, sizeof(u32), (void*) (state->auto_tss_key_address + state->config.offsets.py_tss_t_offsets.is_initialized));
        if (err != 0) {
            BPF_TRACE("python: failed to read is_initialized at address %p: %d", state->auto_tss_key_address + state->config.offsets.py_tss_t_offsets.is_initialized, err);
            return false;
        }

        if (is_initialized == 0) {
            BPF_TRACE("python: tss is not initialized, auto tss key address %p, offset %d", state->auto_tss_key_address, state->config.offsets.py_tss_t_offsets.is_initialized);
            return false;
        }
    }

    u32 extra_offset =  (state->config.offsets.py_tss_t_offsets.key != PYTHON_UNSPECIFIED_OFFSET) ? state->config.offsets.py_tss_t_offsets.key : 0;
    err = bpf_probe_read_user(key_dst, sizeof(i32), (void*) (state->auto_tss_key_address + extra_offset));
    if (err != 0) {
        BPF_TRACE("python: failed to read tss key at address %p: %d", state->auto_tss_key_address + extra_offset, err);
        return false;
    }

    if (*key_dst < 0) {
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

    i32 tss_key = 0;
    if (!read_tss_key(&tss_key, state)) {
        BPF_TRACE("python: failed to read tss key");
        return NULL;
    }

    return pthread_read_tss(&state->pthread_config, (u32) tss_key);
}

static ALWAYS_INLINE void* python_read_py_thread_state_ptr_from_tls(struct python_state* state) {
    if (state == NULL) {
        return NULL;
    }

    void* res = NULL;
    if (state->config.py_thread_state_tls_offset != 0) {
        res = python_read_py_thread_state_ptr_static_tls(state->config.py_thread_state_tls_offset);
    } else if (state->found_pthread_config) {
        res = python_read_py_thread_state_ptr_pthread_tss(state);
    }

    BPF_TRACE("python: read PyThreadState from TLS: %p", res);

    return res;
}

static ALWAYS_INLINE void* python_get_py_thread_state_from_cache(struct python_thread_key* key) {
    if (key == NULL) {
        return NULL;
    }

    void** py_thread_state_ptr = bpf_map_lookup_elem(&python_thread_id_py_thread_state, key);
    if (py_thread_state_ptr == NULL) {
        BPF_TRACE("python: failed to find PyThreadState for pid=%u, thread_id=%u",
                 key->pid, key->thread_id);
        return NULL;
    }

    BPF_TRACE("python: successfully retrieved PyThreadState for pid=%u, thread_id=%u",
             key->pid, key->thread_id);

    return *py_thread_state_ptr;
}

static ALWAYS_INLINE void* python_get_current_thread_state_from_cache(struct python_state* state) {
    if (state == NULL) {
        return NULL;
    }

    state->thread_key.pid = state->pid;

    if (state->config.offsets.py_thread_state_offsets.native_thread_id == PYTHON_UNSPECIFIED_OFFSET) {
        // Note: For x86_64 and glibc 2.4+ (after 2006) pthread_t is actually struct pthread* pointer.
        // Also tcb_head_t is a header of struct pthread.
        state->thread_key.thread_id = get_tcb_pointer();
    } else {
        state->thread_key.thread_id = get_current_task_innermost_tid();
    }

    return python_get_py_thread_state_from_cache(&state->thread_key);
}

static ALWAYS_INLINE u64 python_read_thread_id(void* py_thread_state, struct python_thread_state_offsets* thread_state_offsets) {
    if (py_thread_state == NULL || thread_state_offsets == NULL) {
        return 0;
    }

    u32 offset = thread_state_offsets->native_thread_id;
    if (offset == PYTHON_UNSPECIFIED_OFFSET) {
        offset = thread_state_offsets->thread_id;
    }

    if (offset == PYTHON_UNSPECIFIED_OFFSET) {
        return 0;
    }

    u64 thread_id = 0;
    long err = bpf_probe_read_user(&thread_id, sizeof(u64), (void*)py_thread_state + offset);
    if (err != 0) {
        metric_increment(METRIC_PYTHON_READ_THREAD_ID_ERROR_COUNT);
        BPF_TRACE(
            "python: failed to read thread_id on *PyThreadState at offset %d: %d",
            offset,
            err
        );
        return 0;
    }

    BPF_TRACE("python: read thread_id at offset %d: %u", offset, thread_id);

    return thread_id;
}

static NOINLINE void python_upsert_thread_state(struct python_state* state, void* py_thread_state) {
    if (state == NULL || py_thread_state == NULL) {
        return;
    }

    state->thread_key.pid = state->pid;
    state->thread_key.thread_id = python_read_thread_id(py_thread_state, &state->config.offsets.py_thread_state_offsets);
    if (state->thread_key.thread_id == 0) {
        return;
    }

    long err = bpf_map_update_elem(&python_thread_id_py_thread_state, &state->thread_key, &py_thread_state, BPF_ANY);
    if (err != 0) {
        BPF_TRACE("python: failed to update BPF map with thread_id=%u: %d",
                state->thread_key.thread_id, err);
        return;
    }

    BPF_TRACE("python: successfully upserted PyThreadState %p for thread_id=%u",
            (void*) py_thread_state, state->thread_key.thread_id);
}

static ALWAYS_INLINE void* python_calculate_main_interpreter_state_address(struct python_state* state) {
    if (state == NULL) {
        return NULL;
    }

    if (state->py_runtime_address != 0) {
        return (void*) (state->py_runtime_address + state->config.offsets.py_runtime_state_offsets.py_interpreters_main);
    }

    return (void*) state->py_interp_head_address;
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
    struct python_state* state
) {
    if (state == NULL) {
        return NULL;
    }

    void* main_interpreter_state_address = python_calculate_main_interpreter_state_address(state);
    if (main_interpreter_state_address == NULL) {
        return NULL;
    }
    void* main_interpreter_state = NULL;
    long err = bpf_probe_read_user(&main_interpreter_state, sizeof(void*), main_interpreter_state_address);
    if (err != 0) {
        BPF_TRACE("python: failed to read main *PyInterpreterState: %d", err);
        return NULL;
    }

    void* head_thread_state = python_retrieve_thread_state_from_interpreterstate(main_interpreter_state, &state->config.offsets.py_interpreter_state_offsets);

    BPF_TRACE("python: retrieved head *PyThreadState from *PyInterpreterState %p", head_thread_state);

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
    void* some_thread_state = python_read_py_thread_state_ptr_from_tls(state);
    if (some_thread_state == NULL) {
        some_thread_state = python_get_head_thread_state(state);
    }

    python_fill_threads_cache(state, some_thread_state);

    void* current_thread_state = python_get_current_thread_state_from_cache(state);
    if (current_thread_state == NULL) {
        BPF_TRACE("python: failed to retrieve PyThreadState from both TLS and cache for thread");
    }

    BPF_TRACE("python: successfully retrieved PyThreadState %p", current_thread_state);

    return current_thread_state;
}
