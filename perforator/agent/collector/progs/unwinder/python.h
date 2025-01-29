#pragma once

#include "binary.h"
#include "metrics.h"
#include "process.h"
#include "py_threads.h"

#include <bpf/bpf.h>

#include <stddef.h>

enum {
    PYTHON_MAX_STACK_DEPTH = 128,
    PYTHON_MAX_FUNCNAME_LENGTH = 256,
    PYTHON_MAX_FILENAME_LENGTH = 256,
    MAX_PYTHON_SYMBOLS_SIZE = 200000,
    PYTHON_CFRAME_LINENO_ID = -1,
    PYTHON_UNSPECIFIED_OFFSET = -1,
};

enum python_frame_owner : u8 {
    FRAME_OWNED_BY_THREAD = 0,
    FRAME_OWNED_BY_GENERATOR = 1,
    FRAME_OWNED_BY_FRAME_OBJECT = 2,
    FRAME_OWNED_BY_CSTACK = 3,
};

struct python_ascii_object_offsets {
    u32 length_offset;
    u32 data_offset;
    u32 state_offset;
    u8 ascii_bit;
    u8 compact_bit;
    u8 statically_allocated_bit;
};

struct python_code_object_offsets {
    u32 co_firstlineno_offset;
    u32 filename_offset;
    u32 qualname_offset;
};

struct python_interpreter_frame_offsets {
    u32 f_code_offset;
    u32 previous_offset;
    u32 owner_offset;
};

struct python_cframe_offsets {
    u32 current_frame_offset;
};

struct python_internals_offsets {
    struct python_runtime_state_offsets py_runtime_state_offsets;
    struct python_thread_state_offsets py_thread_state_offsets;
    struct python_cframe_offsets py_cframe_offsets;
    struct python_interpreter_frame_offsets py_interpreter_frame_offsets;
    struct python_interpreter_state_offsets py_interpreter_state_offsets;
    struct python_code_object_offsets py_code_object_offsets;
    struct python_ascii_object_offsets py_ascii_object_offsets;
};

struct python_config {
    u64 py_thread_state_tls_offset;
    u64 py_runtime_relative_address;
    u32 version;

    struct python_internals_offsets offsets;
};

// hope this is enough to avoid collisions
// code_objects are usually allocated once,
//   so this should be good enough identifier within the process.
// Though add co_firstlineno which is quite granular
struct python_symbol_key {
    u64 code_object;
    u32 pid;
    int co_firstlineno;
};

struct python_symbol {
    char file_name[PYTHON_MAX_FILENAME_LENGTH];
    char qual_name[PYTHON_MAX_FUNCNAME_LENGTH];
};

struct python_code_object {
    u64 filename;
    u64 qualname;
};

struct python_frame {
    struct python_symbol_key symbol_key;
};

struct python_state {
    struct python_frame frames[PYTHON_MAX_STACK_DEPTH];
    u32 frame_count;
    struct python_symbol symbol;
    struct python_symbol_key symbol_key;
    struct python_code_object code_object;
    u32 pid;
};

BPF_MAP(python_symbols, BPF_MAP_TYPE_LRU_HASH, struct python_symbol_key, struct python_symbol, MAX_PYTHON_SYMBOLS_SIZE);
BPF_MAP(python_storage, BPF_MAP_TYPE_HASH, binary_id, struct python_config, MAX_BINARIES);

static ALWAYS_INLINE void* python_read_current_frame_from_thread_state(struct python_config* config, void* py_thread_state) {
    if (py_thread_state == NULL) {
        return NULL;
    }

    void*  py_thread_state_or_cframe = py_thread_state;
    u32 current_frame_offset = config->offsets.py_thread_state_offsets.current_frame_offset;
    if (config->offsets.py_thread_state_offsets.cframe_offset != PYTHON_UNSPECIFIED_OFFSET) {
        long err = bpf_probe_read_user(&py_thread_state_or_cframe, sizeof(void*), (void*) py_thread_state + config->offsets.py_thread_state_offsets.cframe_offset);
        if (err != 0) {
            metric_increment(METRIC_PYTHON_READ_PYCFRAME_ERROR_COUNT);
            BPF_TRACE(
                "python: failed to read cframe from *Pythread_state by offset %d: %d",
                config->offsets.py_thread_state_offsets.cframe_offset,
                err
            );
            return NULL;
        }
        current_frame_offset = config->offsets.py_cframe_offsets.current_frame_offset;

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
        (void*) frame + config->offsets.py_interpreter_frame_offsets.previous_offset
    );
    if (err != 0) {
        metric_increment(METRIC_PYTHON_READ_PREVIOUS_FRAME_ERROR);
        BPF_TRACE(
            "python: failed to read previous frame by offset %d: %d",
            config->offsets.py_interpreter_frame_offsets.previous_offset,
            err
        );
        return NULL;
    }

    return frame;
}

static ALWAYS_INLINE bool python_read_frame_owner(enum python_frame_owner* owner, void* frame, struct python_config* config) {
    long err = bpf_probe_read_user(owner, sizeof(u8), (void*) frame + config->offsets.py_interpreter_frame_offsets.owner_offset);
    if (err != 0) {
        metric_increment(METRIC_PYTHON_READ_FRAME_OWNER_ERROR_COUNT);
        BPF_TRACE(
            "python: failed to read frame owner at offset %d: %d",
            config->offsets.py_interpreter_frame_offsets.owner_offset,
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
    state->code_object.qualname = 0;
}

static ALWAYS_INLINE bool python_read_code_object(struct python_code_object* result_object, struct python_config* config, void* code) {
    if (result_object == NULL || config == NULL) {
        return false;
    }
    result_object->filename = 0;
    result_object->qualname = 0;

    long err = bpf_probe_read_user(&result_object->qualname, sizeof(void*), code + config->offsets.py_code_object_offsets.qualname_offset);
    if (err != 0) {
        BPF_TRACE(
            "python: failed to read qualname at offset %d: %d",
            config->offsets.py_code_object_offsets.qualname_offset,
            err
        );
        return false;
    }

    err = bpf_probe_read_user(&result_object->filename, sizeof(void*), code + config->offsets.py_code_object_offsets.filename_offset);
    if (err != 0) {
        BPF_TRACE(
            "python: failed to read filename at offset %d: %d",
            config->offsets.py_code_object_offsets.filename_offset,
            err
        );
        return false;
    }

    BPF_TRACE("python: read filename and qualname pointers: %p, %p", result_object->filename, result_object->qualname);

    return true;
}

static ALWAYS_INLINE bool python_read_python_ascii_string(char* buffer, size_t buffer_size, struct python_config* config, void* py_object) {
    if (buffer == NULL || buffer_size <= 0) {
        return false;
    }
    buffer[0] = '\0';

    size_t length = 0;
    long err = bpf_probe_read_user(&length, sizeof(length), (void*) py_object + config->offsets.py_ascii_object_offsets.length_offset);
    if (err != 0) {
        BPF_TRACE("python: failed to read ascii string length: %d", err);
        return false;
    }
    ++length;

    BPF_TRACE("python: read ascii string length %d, buffer_size %u", length, buffer_size);

    u32 status = 0;
    err = bpf_probe_read_user(&status, sizeof(u32), (void*) py_object + config->offsets.py_ascii_object_offsets.state_offset);
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

    err = bpf_probe_read_user_str(buffer, length, (void*) py_object + config->offsets.py_ascii_object_offsets.data_offset);
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

    if (code_object->qualname != 0) {
        if (!python_read_python_ascii_string(result_symbol->qual_name, sizeof(result_symbol->qual_name), config, (void*) code_object->qualname)) {
            BPF_TRACE("python: failed to read code object qualname");
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
    long err = bpf_probe_read_user(&code, sizeof(void*), (void*) frame + config->offsets.py_interpreter_frame_offsets.f_code_offset);
    if (err != 0) {
        BPF_TRACE(
            "python: failed to read PyCodeObject* at offset %d: %d",
            config->offsets.py_interpreter_frame_offsets.f_code_offset,
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
    err = bpf_probe_read(&state->symbol_key.co_firstlineno, sizeof(int), (void*) code + config->offsets.py_code_object_offsets.co_firstlineno_offset);
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
    void* py_interpreter_frame,
    struct python_config* config,
    struct python_state* state
) {
    if (config == NULL || state == NULL) {
        return;
    }

    for (int i = 0; i < PYTHON_MAX_STACK_DEPTH; i++) {
        if (py_interpreter_frame == NULL) {
            break;
        }

        enum python_frame_owner owner = FRAME_OWNED_BY_THREAD;
        if (!python_read_frame_owner(&owner, py_interpreter_frame, config)) {
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

        if (!python_process_frame(&state->frames[i], py_interpreter_frame, config, state)) {
            break;
        }
        state->frame_count = i + 1;

        BPF_TRACE("python: Successfully processed frame %d", i);

move_to_next_frame:
        py_interpreter_frame = python_read_previous_frame(py_interpreter_frame, config);
    }

    BPF_TRACE("python: Collected %d frames", state->frame_count);
}

static ALWAYS_INLINE void python_collect_stack(
    struct process_info* proc_info,
    struct python_state* state
) {
    if (proc_info == NULL || state == NULL) {
        return;
    }

    // TODO(@pashaguskov): support libpython.so
    binary_id id = proc_info->main_binary_id;
    struct python_config* config = bpf_map_lookup_elem(&python_storage, &id);
    if (config == NULL) {
        return;
    }

    metric_increment(METRIC_PYTHON_PROCESSED_STACKS_COUNT);

    void* py_thread_state_addr = python_get_thread_state_and_update_cache(
        config->py_thread_state_tls_offset,
        config->py_runtime_relative_address,
        &config->offsets.py_runtime_state_offsets,
        &config->offsets.py_interpreter_state_offsets,
        &config->offsets.py_thread_state_offsets
    );
    if (py_thread_state_addr == NULL) {
        metric_increment(METRIC_PYTHON_TLS_THREAD_STATE_NULL);
        BPF_TRACE("python: read NULL *Pythread_state");
        return;
    }

    BPF_TRACE("python: Successfully extracted Pythread_state addr %p", py_thread_state_addr);

    void* py_interpreter_frame = python_read_current_frame_from_thread_state(config, py_thread_state_addr);
    if (py_interpreter_frame == NULL) {
        return;
    }

    BPF_TRACE("python: Successfully read PyInterpreterFrame addr %p", py_interpreter_frame);

    python_reset_state(state);
    python_walk_stack(py_interpreter_frame, config, state);
}
