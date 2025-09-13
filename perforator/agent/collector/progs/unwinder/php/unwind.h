#pragma once

#include "../process.h"
#include "types.h"
#include "trace.h"

#ifdef PERFORATOR_ENABLE_PHP

static ALWAYS_INLINE bool php_retrieve_configs(
    struct process_info* proc_info,
    struct php_state* state
) {
    if (proc_info == NULL || state == NULL) {
        return false;
    }

    binary_id id = proc_info->php_binary.id;
    struct php_config* config = bpf_map_lookup_elem(&php_storage, &id);
    if (config == NULL) {
        return false;
    }

    state->config = *config;
    if (config->executor_globals_elf_vaddr != 0) {
        state->executor_globals_mem_vaddr = proc_info->php_binary.base_address + config->executor_globals_elf_vaddr;
    }

    return true;
}

static ALWAYS_INLINE void* read_execute_data(u64 executor_globals_mem_vaddr, u32 execute_data_offset) {
    u64 execute_data_addr = executor_globals_mem_vaddr + execute_data_offset;
    void* execute_data;
    long err = bpf_probe_read_user(&execute_data, sizeof(void*), (void*) execute_data_addr);
    if (err != 0) {
        metric_increment(METRIC_PHP_READ_EXECUTE_DATA_ERROR_COUNT);
        PHP_TRACE(
            "failed to read execute data on address %lld: %d",
            execute_data_addr,
            err
        );
        return NULL;
    }

    return execute_data;
}

static ALWAYS_INLINE void php_reset_state(struct php_state* state) {
    if (state == NULL) {
        return;
    }
    state->frame_count = 0;
    state->executor_globals_mem_vaddr = 0;
    state->execute_data = 0;
}

static ALWAYS_INLINE bool php_read_string(char* buf, size_t buf_size, u8* res_length, struct php_config* config, void* zend_string_addr) {
    if (buf == NULL || buf_size == 0 || res_length == NULL || config == NULL || zend_string_addr == NULL) {
        return false;
    }

    size_t length = 0;
    long err = bpf_probe_read_user(&length, sizeof(length), (void*) zend_string_addr + config->offsets.zend_string.len);
    if (err != 0) {
        PHP_TRACE("failed to read string length: %d", err);
        return false;
    }

    if (length == 0) {
        PHP_TRACE("string len is 0");
        return false;
    }

    PHP_TRACE("read string length %d, buf_size %u", length, buf_size);

    if (length >= buf_size) {
        length = buf_size - 1;
    }
    length &= INTERPRETER_SYMBOL_STRING_LENGTH_VERIFIER_MASK;

    // On success, the strictly positive length of the output string,
    // including the trailing NULL character. On error, a negative value.
    err = bpf_probe_read_user_str(buf, length + 1, (void*) zend_string_addr + config->offsets.zend_string.val);
    if (err <= 0) {
        PHP_TRACE("failed to read string data: %d", err);
        return false;
    }

    *res_length = err - 1;
    PHP_TRACE("Successfully read string of length %d", *res_length);
    return true;
}

static ALWAYS_INLINE bool php_read_function_data(struct php_state* state, void* zend_function, u8 function_type) {
    if (state == NULL || zend_function == NULL) {
        return false;
    }

    state->function_data.filename_addr = 0;
    state->function_data.funcname_addr = 0;

    struct php_config* config = &state->config;

    long err = bpf_probe_read_user(&state->function_data.funcname_addr, sizeof(void*),
                                   zend_function + config->offsets.function.common_funcname);
    if (err != 0) {
        PHP_TRACE("failed to read function name addr: %d", err);
        return false;
    }

    // Read filename for user functions
    if (function_type == ZEND_USER_FUNCTION || function_type == ZEND_EVAL_CODE) {
        err = bpf_probe_read_user(&state->function_data.filename_addr, sizeof(void*),
                                  zend_function + config->offsets.function.op_array.filename);
        if (err != 0) {
            PHP_TRACE("failed to read filename addr: %d", err);
            return false;
        }
    }

    PHP_TRACE("read function name and filename pointers: %p, %p, type: %d",
          state->function_data.funcname_addr,
          state->function_data.filename_addr,
          function_type
    );

    return true;
}

static ALWAYS_INLINE bool php_read_symbol(struct php_state* state) {
    if (state == NULL) {
        return false;
    }

    struct php_config* config = &state->config;
    state->symbol.codepoint_size = 1;
    char* buf = state->symbol.data;

    if (!php_read_string(buf, sizeof(state->symbol.data), &state->symbol.name_length,
                         config, (void*) state->function_data.funcname_addr)) {
        PHP_TRACE("failed to read function name");
        return false;
    }

    state->symbol.name_length &= INTERPRETER_SYMBOL_STRING_LENGTH_VERIFIER_MASK;

    state->symbol.filename_length = 0;
    buf = state->symbol.data + state->symbol.name_length;
    if (!php_read_string(buf, sizeof(state->symbol.data) - state->symbol.name_length,
                            &state->symbol.filename_length, config,
                            (void*) state->function_data.filename_addr)) {
        PHP_TRACE("failed to read filename");
        // filename is optional
    }

    return true;
}

static ALWAYS_INLINE bool php_process_frame(struct php_state* state, struct php_config* config, struct interpreter_frame* frame, void* execute_data, u32* type_info) {
    if (state == NULL || execute_data == NULL) {
        return false;
    }

    void* zend_function;
    long err = bpf_probe_read_user(
          &zend_function, sizeof(void*), execute_data + config->offsets.execute_data.function);
    if (err != 0) {
        metric_increment(METRIC_PHP_READ_ZEND_FUNCTION_ERROR_COUNT);
        PHP_TRACE(
            "failed to read zend function on execute_data address %p with offset %d: %d",
            execute_data,
            config->offsets.execute_data.function,
            err
        );
        return false;
    }

    if (zend_function == NULL) {
        *frame = PHP_FRAME_UNKNOWN;
        return true;
    }

    u8 function_type;
    err = bpf_probe_read_user(
          &function_type, sizeof(function_type), zend_function + config->offsets.function.type);
    if (err != 0) {
        metric_increment(METRIC_PHP_READ_FUNCTION_TYPE_ERROR_COUNT);
        PHP_TRACE(
            "failed to read function type on zend_function address %p with offset %d: %d",
            zend_function,
            config->offsets.function.type,
            err
        );
        return false;
    }

    u32 linestart = 0;

    if (function_type == ZEND_USER_FUNCTION || function_type == ZEND_EVAL_CODE) {
        err = bpf_probe_read_user(
              type_info, sizeof(u32), execute_data + config->offsets.execute_data.this_type_info);
        if (err != 0) {
            metric_increment(METRIC_PHP_READ_TYPE_INFO_ERROR_COUNT);
            PHP_TRACE("failed to read type info on execute_data address %p with offset %d: %d",
                execute_data,
                config->offsets.execute_data.this_type_info,
                err
            );
            return false;
        }

        err = bpf_probe_read_user(&linestart, sizeof(u32),
                                  zend_function + config->offsets.function.op_array.linestart);
        if (err != 0) {
            PHP_TRACE("failed to read linestart: %d", err);
            linestart = 0;
        }
    }

    state->symbol_key.pid = state->pid;
    state->symbol_key.object_addr = (u64)zend_function;
    state->symbol_key.linestart = linestart;

    frame->symbol_key = state->symbol_key;

    struct php_symbol* symbol = bpf_map_lookup_elem(&interpreter_symbols, &state->symbol_key);
    if (symbol != NULL) {
        PHP_TRACE("already saved this symbol pid: %u, function_addr %p, linestart: %u",
                  state->symbol_key.pid, state->symbol_key.object_addr, state->symbol_key.linestart);
        return true;
    }

    if (!php_read_function_data(state, zend_function, function_type)) {
        return false;
    }

    if (!php_read_symbol(state)) {
        metric_increment(METRIC_PHP_FAILED_TO_READ_SYMBOL_COUNT);
        return false;
    }

    err = bpf_map_update_elem(&interpreter_symbols, &state->symbol_key, &state->symbol, BPF_ANY);
    if (err != 0) {
        PHP_TRACE("failed to update php symbol: %d", err);
    }

    return true;
}

static ALWAYS_INLINE void* read_prev_execute_data(struct php_config* config, void* execute_data) {
    if (execute_data == NULL) {
        return NULL;
    }
    void* prev_execute_data;
    long err = bpf_probe_read_user(
          &prev_execute_data, sizeof(void*), execute_data + config->offsets.execute_data.prev_execute_data);
    if (err != 0) {
        return NULL;
    }

    return prev_execute_data;
}

static ALWAYS_INLINE void php_walk_stack(struct php_state* state, void* execute_data) {
    if (state == NULL) {
        return;
    }

    u32 type_info = 0;
    struct php_config* config = &state->config;

    for (int i = 0; i < PHP_MAX_STACK_DEPTH; i++) {
        if (execute_data == NULL) {
            break;
        }

        if (!php_process_frame(state, config, &state->frames[i], execute_data, &type_info)) {
            break;
        }

        state->frame_count = i + 1;
        PHP_TRACE("Successfully processed frame %d", i);

        if (type_info & ZEND_TOP_FUNCTION) {
            break;
        }

        execute_data = read_prev_execute_data(config, execute_data);
    }

    PHP_TRACE("Collected %d frames", state->frame_count);
}

static ALWAYS_INLINE void php_collect_stack(
    struct process_info* proc_info,
    struct php_state* state
) {
    if (proc_info == NULL || state == NULL || !is_mapped(proc_info->php_binary)) {
        return;
    }

    bool found_config = php_retrieve_configs(proc_info, state);
    if (!found_config) {
        return;
    }

    metric_increment(METRIC_PHP_PROCESSED_STACKS_COUNT);

    struct php_config* config = &state->config;
    void* execute_data = read_execute_data(state->executor_globals_mem_vaddr, config->offsets.zend_execute_data);

    // TODO: Check JIT
    php_reset_state(state);
    php_walk_stack(state, execute_data);

    return;
}

#endif
