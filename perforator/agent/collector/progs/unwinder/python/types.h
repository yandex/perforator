#pragma once

#include "../binary.h"
#include "../interpreter/types.h"
#include "../pthread.h"

#define PYVERSION(major, minor, micro) \
    (((u32)(major) << 16) | ((u32)(minor) << 8) | ((u32)(micro)))

struct python_interpreter_state_offsets {
    u32 next;
    u32 threads_head;
};

struct python_runtime_state_offsets {
    u32 py_interpreters_main;
};

struct python_thread_state_offsets {
    u32 cframe;
    u32 current_frame;
    u32 thread_id; // pthread thread id
    u32 native_thread_id; // OS thread id
    u32 prev_thread;
    u32 next_thread;
};

struct python_thread_key {
    // This field stores one of these options:
    // - PyThreadState->native_thread_id which is innermost namespace native thread id (pid_t)
    //   in case we are dealing with CPython 3.11+
    // - PyThreadState->thread_id which is pthread thread id (pthread_t) for cpython before 3.11:
    //   in case we are dealing with CPython 3.10-
    // This field is only unique within a process.
    u64 thread_id;
    u32 pid;
};

struct python_code_object {
    u64 filename;
    u64 name;
};

enum {
    MAX_PYTHON_THREADS = 16384,
    MAX_PYTHON_THREAD_STATE_WALK = 32,
    PYTHON_MAX_STACK_DEPTH = 128,
    PYTHON_CFRAME_LINENO_ID = -1,
    PYTHON_UNSPECIFIED_OFFSET = -1,
};

enum python_frame_owner : u8 {
    FRAME_OWNED_BY_THREAD = 0,
    FRAME_OWNED_BY_GENERATOR = 1,
    FRAME_OWNED_BY_FRAME_OBJECT = 2,
    FRAME_OWNED_BY_CSTACK = 3,
};

// This struct stores offsets for:
// CPython 3.3+ - PyASCIIObject
// CPython 3.0-3.2 - PyUnicodeObject
// CPython 2 - PyStringObject
// PyStringObject has the same layout as PyASCIIObject (but different field names), PyUnicodeObject is different.
struct python_string_object_offsets {
    // These fields are present for PyUnicodeObject, PyASCIIObject and PyStringObject
    u32 length;
    u32 data;

    // These fields are only present for PyASCIIObject
    u32 state;
    u8 ascii_bit;
    u8 compact_bit;
    u8 statically_allocated_bit;
};

struct python_code_object_offsets {
    u32 co_firstlineno;
    u32 filename;
    u32 name;
    // co_code_adaptive is the start of the inlined bytecode array within PyCodeObject
    // (CPython 3.11+). Used in userspace to convert instruction pointer into a
    // bytecode offset: offset = instr_ptr - (code_addr + co_code_adaptive).
    u32 co_code_adaptive;
    // co_linetable is the offset of the PyBytesObject* line table field within
    // PyCodeObject (CPython 3.11+). Read in userspace via process_vm_readv.
    u32 co_linetable;
};

struct python_frame_object_offsets {
    u32 f_code;
    u32 f_back;
};

struct python_frame_offsets {
    u32 f_code;
    u32 previous;
    u32 owner;
    // instr_ptr is the offset of the currently executing bytecode instruction pointer
    // within _PyInterpreterFrame. Sourced from `prev_instr` on CPython 3.11/3.12
    // and from `instr_ptr` on CPython 3.13+.
    u32 instr_ptr;
};

// python_bytes_object_offsets describes the layout of PyBytesObject. Used by
// userspace to read the co_linetable bytes after dereferencing co_linetable.
struct python_bytes_object_offsets {
    u32 ob_size;
    u32 ob_sval;
};

struct python_cframe_offsets {
    u32 current_frame;
};

struct python_tss_t_offsets {
    u32 is_initialized;
    u32 key;
};

struct python_internals_offsets {
    struct python_runtime_state_offsets py_runtime_state_offsets;
    struct python_thread_state_offsets py_thread_state_offsets;
    struct python_cframe_offsets py_cframe_offsets;
    struct python_frame_offsets py_frame_offsets;
    struct python_interpreter_state_offsets py_interpreter_state_offsets;
    struct python_code_object_offsets py_code_object_offsets;
    struct python_string_object_offsets py_string_object_offsets;
    struct python_tss_t_offsets py_tss_t_offsets;
    struct python_bytes_object_offsets py_bytes_object_offsets;
};

struct python_config {
    u64 py_thread_state_tls_offset;
    u64 py_runtime_relative_address;
    u64 py_interp_head_relative_address;
    u64 auto_tss_key_relative_address;
    u32 version;
    u32 unicode_type_size_log2;

    struct python_internals_offsets offsets;
};

struct python_state {
    struct python_thread_key thread_key;
    struct python_config config;
    struct pthread_config pthread_config;
    bool found_pthread_config;
    u64 py_runtime_address;
    u64 py_interp_head_address;
    u64 auto_tss_key_address;
    struct interpreter_frame frames[PYTHON_MAX_STACK_DEPTH];
    u32 frame_count;
    struct symbol symbol;
    struct symbol_key symbol_key;
    struct python_code_object code_object;
    u32 pid;
};

BPF_MAP(python_thread_id_py_thread_state, BPF_MAP_TYPE_LRU_HASH, struct python_thread_key, void*, MAX_PYTHON_THREADS);
BPF_MAP(python_storage, BPF_MAP_TYPE_HASH, binary_id, struct python_config, MAX_BINARIES);
