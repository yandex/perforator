#pragma once

#include "binary.h"
#include "metrics.h"
#include "process.h"
#include "pthread.h"
#include "py_types.h"
#include "py_threads.h"

#include <bpf/bpf.h>

#include <stddef.h>

static ALWAYS_INLINE void* python_read_current_frame_from_thread_state(struct python_config* config, void* py_thread_state) {
    if (py_thread_state == NULL || config == NULL) {
        return NULL;
    }

    void* py_thread_state_or_cframe = py_thread_state;
    u32 current_frame_offset = config->offsets.py_thread_state_offsets.current_frame;
    if (config->offsets.py_thread_state_offsets.cframe != PYTHON_UNSPECIFIED_OFFSET) {
        long err = bpf_probe_read_user(&py_thread_state_or_cframe, sizeof(void*), (void*) py_thread_state + config->offsets.py_thread_state_offsets.cframe);
        if (err != 0) {
            metric_increment(METRIC_PYTHON_READ_PYCFRAME_ERROR_COUNT);
            BPF_TRACE(
                "python: failed to read cframe from *Pythread_state by offset %d: %d",
                config->offsets.py_thread_state_offsets.cframe,
                err
            );
            return NULL;
        }
        current_frame_offset = config->offsets.py_cframe_offsets.current_frame;

        BPF_TRACE("python: Successfully read *_PyCFrame addr %p", py_thread_state_or_cframe);
    }

    if (py_thread_state_or_cframe == NULL) {
        metric_increment(METRIC_PYTHON_PYCFRAME_NULL);
        return NULL;
    }

    void *frame = NULL;
    long err = bpf_probe_read_user(&frame, sizeof(void*), (void*) py_thread_state_or_cframe + current_frame_offset);
    if (err != 0) {
        metric_increment(METRIC_PYTHON_READ_PY_INTERPRETER_FRAME_ERROR_COUNT);
        BPF_TRACE(
            "python: failed to read current_frame by offset %d from addr %p: %d",
            current_frame_offset,
            (void*) py_thread_state_or_cframe + current_frame_offset,
            err
        );
        return NULL;
    }

    if (frame == NULL) {
        metric_increment(METRIC_PYTHON_PY_INTERPRETER_FRAME_NULL);
    }

    return frame;
}

static ALWAYS_INLINE void* python_read_previous_frame(void* frame, struct python_config* config) {
    long err = bpf_probe_read_user(
        &frame,
        sizeof(void*),
        (void*) frame + config->offsets.py_frame_offsets.previous
    );
    if (err != 0) {
        metric_increment(METRIC_PYTHON_READ_PREVIOUS_FRAME_ERROR);
        BPF_TRACE(
            "python: failed to read previous frame by offset %d: %d",
            config->offsets.py_frame_offsets.previous,
            err
        );
        return NULL;
    }

    return frame;
}

static ALWAYS_INLINE bool python_read_frame_owner(enum python_frame_owner* owner, void* frame, struct python_config* config) {
    if (config->offsets.py_frame_offsets.owner == PYTHON_UNSPECIFIED_OFFSET) {
        // Before Python 3.11.4
        *owner = FRAME_OWNED_BY_THREAD;
        return true;
    }

    // For Python versions (3.11.4+) with owner field, read it
    long err = bpf_probe_read_user(owner, sizeof(u8), (void*) frame + config->offsets.py_frame_offsets.owner);
    if (err != 0) {
        metric_increment(METRIC_PYTHON_READ_FRAME_OWNER_ERROR_COUNT);
        BPF_TRACE(
            "python: failed to read frame owner at offset %d: %d",
            config->offsets.py_frame_offsets.owner,
            err
        );
        return false;
    }

    return true;
}

static ALWAYS_INLINE void python_reset_state(struct python_state* state) {
    if (state == NULL) {
        return;
    }

    state->frame_count = 0;
    state->code_object.filename = 0;
    state->code_object.name = 0;
}

static ALWAYS_INLINE bool python_read_code_object(struct python_code_object* result_object, struct python_config* config, void* code) {
    if (result_object == NULL || config == NULL) {
        return false;
    }
    result_object->filename = 0;
    result_object->name = 0;

    long err = bpf_probe_read_user(&result_object->name, sizeof(void*), code + config->offsets.py_code_object_offsets.name);
    if (err != 0) {
        BPF_TRACE(
            "python: failed to read name at offset %d: %d",
            config->offsets.py_code_object_offsets.name,
            err
        );
        return false;
    }

    err = bpf_probe_read_user(&result_object->filename, sizeof(void*), code + config->offsets.py_code_object_offsets.filename);
    if (err != 0) {
        BPF_TRACE(
            "python: failed to read filename at offset %d: %d",
            config->offsets.py_code_object_offsets.filename,
            err
        );
        return false;
    }

    BPF_TRACE("python: read filename and name pointers: %p, %p", result_object->filename, result_object->name);

    return true;
}

static ALWAYS_INLINE bool python_read_python_ascii_string(char* buffer, size_t buffer_size, struct python_config* config, void* py_object) {
    if (buffer == NULL || buffer_size <= 0) {
        return false;
    }
    buffer[0] = '\0';

    size_t length = 0;
    long err = bpf_probe_read_user(&length, sizeof(length), (void*) py_object + config->offsets.py_ascii_object_offsets.length);
    if (err != 0) {
        BPF_TRACE("python: failed to read ascii string length: %d", err);
        return false;
    }
    ++length;

    BPF_TRACE("python: read ascii string length %d, buffer_size %u", length, buffer_size);

    u32 status = 0;
    err = bpf_probe_read_user(&status, sizeof(u32), (void*) py_object + config->offsets.py_ascii_object_offsets.state);
    if (err != 0) {
        BPF_TRACE("python: failed to read ascii status: %d", err);
        return false;
    }

    if ((status & (1 << config->offsets.py_ascii_object_offsets.ascii_bit)) == 0
        || (status & (1 << config->offsets.py_ascii_object_offsets.compact_bit)) == 0) {
        metric_increment(METRIC_PYTHON_NON_ASCII_COMPACT_STRINGS_COUNT);
        return false;
    }

    BPF_TRACE("python: found string status %u", ((status << 24) >> 24));

    if (length > buffer_size) {
        length = buffer_size;
    }

    err = bpf_probe_read_user_str(buffer, length, (void*) py_object + config->offsets.py_ascii_object_offsets.data);
    if (err < 0)  {
        BPF_TRACE("python: failed to read ascii string data: %d", err);
        return false;
    }

    BPF_TRACE("python: Successfully read ASCII string of length %d", err);

    return true;
}

static ALWAYS_INLINE bool python_read_symbol(struct python_symbol* result_symbol, struct python_config* config, struct python_code_object* code_object) {
    if (result_symbol == NULL || code_object == NULL || config == NULL) {
        return false;
    }

    if (code_object->filename != 0) {
        if (!python_read_python_ascii_string(result_symbol->file_name, sizeof(result_symbol->file_name), config, (void*) code_object->filename)) {
            BPF_TRACE("python: failed to read code object filename");
            return false;
        }
    }

    if (code_object->name != 0) {
        if (!python_read_python_ascii_string(result_symbol->name, sizeof(result_symbol->name), config, (void*) code_object->name)) {
            BPF_TRACE("python: failed to read code object name");
            return false;
        }
    }

    return true;
}

static ALWAYS_INLINE bool python_process_frame(struct python_frame* res_frame, void* frame, struct python_config* config, struct python_state* state) {
    if (config == NULL || state == NULL || res_frame == NULL) {
        return false;
    }

    void* code = NULL;
    long err = bpf_probe_read_user(&code, sizeof(void*), (void*) frame + config->offsets.py_frame_offsets.f_code);
    if (err != 0) {
        BPF_TRACE(
            "python: failed to read PyCodeObject* at offset %d: %d",
            config->offsets.py_frame_offsets.f_code,
            err
        );
        return false;
    }

    if (code == NULL) {
        BPF_TRACE("python: read NULL PyCodeObject*");
        return false;
    }

    state->symbol_key.pid = state->pid;
    state->symbol_key.code_object = (u64) code;
    err = bpf_probe_read(&state->symbol_key.co_firstlineno, sizeof(int), (void*) code + config->offsets.py_code_object_offsets.co_firstlineno);
    if (err != 0) {
        BPF_TRACE("python: failed to read co_firstlineno: %d", err);
        return false;
    }

    res_frame->symbol_key = state->symbol_key;

    struct python_symbol* symbol = bpf_map_lookup_elem(&python_symbols, &state->symbol_key);
    if (symbol != NULL) {
        BPF_TRACE(
            "python: already saved this symbol pid: %u, code_object %p, first line: %d",
            state->symbol_key.pid,
            state->symbol_key.code_object,
            state->symbol_key.co_firstlineno
        );
        return true;
    }

    if (!python_read_code_object(&state->code_object, config, code)) {
        return false;
    }

    if (!python_read_symbol(&state->symbol, config, &state->code_object)) {
        return false;
    }

    err = bpf_map_update_elem(&python_symbols, &state->symbol_key, &state->symbol, BPF_ANY);
    if (err != 0) {
        BPF_TRACE("python: failed to update python symbol: %d", err);
    }

    return true;
}

static ALWAYS_INLINE void python_walk_stack(
    void* py_frame,
    struct python_config* config,
    struct python_state* state
) {
    if (config == NULL || state == NULL) {
        return;
    }

    for (int i = 0; i < PYTHON_MAX_STACK_DEPTH; i++) {
        if (py_frame == NULL) {
            break;
        }

        enum python_frame_owner owner = FRAME_OWNED_BY_THREAD;
        if (!python_read_frame_owner(&owner, py_frame, config)) {
            break;
        }

        if (owner == FRAME_OWNED_BY_CSTACK) {
            // stub frame in case python is called from C code.
            //  2 consecutive frames must not be owned by C stack.
            BPF_TRACE("python: frame owned by c stack");
            state->frames[i].symbol_key.co_firstlineno = PYTHON_CFRAME_LINENO_ID;
            state->frames[i].symbol_key.pid = 0;
            state->frames[i].symbol_key.code_object = 0;
            state->frame_count = i + 1;
            goto move_to_next_frame;
        }

        if (!python_process_frame(&state->frames[i], py_frame, config, state)) {
            break;
        }
        state->frame_count = i + 1;

        BPF_TRACE("python: Successfully processed frame %d", i);

move_to_next_frame:
        py_frame = python_read_previous_frame(py_frame, config);
    }

    BPF_TRACE("python: Collected %d frames", state->frame_count);
}

// Returns true if we should proceed with stack collection
static ALWAYS_INLINE bool python_retrieve_configs(
    struct process_info* proc_info,
    struct python_state* state
) {
    if (proc_info == NULL || state == NULL) {
        return false;
    }

    binary_id id = proc_info->interpreter_binary.id;
    struct python_config* config = bpf_map_lookup_elem(&python_storage, &id);
    if (config == NULL) {
        return false;
    }
    state->config = *config;
    if (config->py_runtime_relative_address != 0) {
        state->py_runtime_address = proc_info->interpreter_binary.start_address + config->py_runtime_relative_address;
    }
    if (config->auto_tss_key_relative_address != 0) {
        state->auto_tss_key_address = proc_info->interpreter_binary.start_address + config->auto_tss_key_relative_address;
    }

    if (proc_info->pthread_binary.type != SPECIAL_BINARY_TYPE_PTHREAD_GLIBC) {
        return true;
    }

    id = proc_info->pthread_binary.id;
    struct pthread_config* pthread_config = bpf_map_lookup_elem(&pthread_storage, &id);
    if (pthread_config != NULL) {
        state->pthread_config = *pthread_config;
    }
    state->found_pthread_config = (pthread_config != NULL);

    return (config != NULL);
}

static ALWAYS_INLINE void python_collect_stack(
    struct process_info* proc_info,
    struct python_state* state
) {
    if (proc_info == NULL || state == NULL) {
        return;
    }

    if (proc_info->interpreter_binary.type != SPECIAL_BINARY_TYPE_PYTHON_INTERPRETER) {
        return;
    }

    bool found_config = python_retrieve_configs(proc_info, state);
    if (!found_config) {
        return;
    }

    metric_increment(METRIC_PYTHON_PROCESSED_STACKS_COUNT);

    void* py_thread_state_addr = python_get_thread_state_and_update_cache(state);
    if (py_thread_state_addr == NULL) {
        metric_increment(METRIC_PYTHON_TLS_THREAD_STATE_NULL);
        BPF_TRACE("python: read NULL *PyThreadState");
        return;
    }

    BPF_TRACE("python: Successfully extracted PyThreadState addr %p", py_thread_state_addr);

    void* py_frame = python_read_current_frame_from_thread_state(&state->config, py_thread_state_addr);
    if (py_frame == NULL) {
        return;
    }

    BPF_TRACE("python: Successfully read python frame addr %p", py_frame);

    python_reset_state(state);
    python_walk_stack(py_frame, &state->config, state);
}
