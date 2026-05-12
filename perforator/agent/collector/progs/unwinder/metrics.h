#pragma once

#include <bpf/types.h>
#include <bpf/bpf.h>

////////////////////////////////////////////////////////////////////////////////

enum metric : u32 {
    METRIC_SAMPLE_COUNT,
    METRIC_SAMPLE_SUCCESSFULL_COUNT,
    METRIC_SAMPLE_UNSUCCESSFULL_COUNT,
    METRIC_EVENT_COUNT,
    METRIC_PERFEVENT_MULTIPLEXED_COUNT,
    METRIC_SIGNALDELIVER_TRIGGERED_COUNT,
    METRIC_SIGNALDELIVER_SAMPLED_COUNT,
    METRIC_STACK_FRAME_DWARF_COUNT,
    METRIC_STACK_FRAME_FP_COUNT,
    METRIC_STACK_FRAME_COUNT,
    METRIC_PROCESS_UNKNOWN_COUNT,
    METRIC_PROCESS_NOTIFIED_COUNT,
    METRIC_FILTERED_KTHREAD_COUNT,
    METRIC_FILTERED_PROCESS_COUNT,
    METRIC_ERROR_STAGE_START_COUNT,
    METRIC_ERROR_STAGE_LOCATETRACEEE_COUNT,
    METRIC_ERROR_STAGE_COLLECTSTACK_COUNT,
    METRIC_ERROR_STAGE_TLS_COUNT,
    METRIC_ERROR_STAGE_COLLECT_PYTHON_STACK_COUNT,
    METRIC_PYTHON_READ_TLS_THREAD_STATE_ERROR_COUNT,
    METRIC_PYTHON_TLS_THREAD_STATE_NULL,
    METRIC_PYTHON_READ_PYCFRAME_ERROR_COUNT,
    METRIC_PYTHON_PYCFRAME_NULL,
    METRIC_PYTHON_READ_PY_INTERPRETER_FRAME_ERROR_COUNT,
    METRIC_PYTHON_PY_INTERPRETER_FRAME_NULL,
    METRIC_PYTHON_READ_PREVIOUS_FRAME_ERROR,
    METRIC_PYTHON_READ_FRAME_OWNER_ERROR_COUNT,
    METRIC_PYTHON_FAILED_TO_READ_SYMBOL_COUNT,
    METRIC_PYTHON_NON_ASCII_COMPACT_STRINGS_COUNT,
    METRIC_PYTHON_PROCESSED_STACKS_COUNT,
    METRIC_PYTHON_READ_THREAD_ID_ERROR_COUNT,
    METRIC_ERROR_STAGE_COLLECT_PHP_STACK_COUNT,
    METRIC_PHP_READ_EXECUTE_DATA_ERROR_COUNT,
    METRIC_PHP_READ_ZEND_FUNCTION_ERROR_COUNT,
    METRIC_PHP_READ_FUNCTION_TYPE_ERROR_COUNT,
    METRIC_PHP_READ_ZEND_OPLINE_ERROR_COUNT,
    METRIC_PHP_READ_LINENO_ERROR_COUNT,
    METRIC_PHP_READ_TYPE_INFO_ERROR_COUNT,
    METRIC_PHP_PROCESSED_STACKS_COUNT,
    METRIC_PHP_FAILED_TO_READ_SYMBOL_COUNT,
    METRIC_ERROR_STAGE_RECORDSAMPLE_COUNT,
    METRIC_ERROR_STAGE_LBR_STACK_COUNT,
    METRIC_DWARF_ERROR_TOOMANYFRAMES_COUNT,
    METRIC_DWARF_ERROR_RULEEVALUATIONFAILED_COUNT,
    METRIC_DWARF_ERROR_NORULEFORINSTRUCTION_COUNT,
    METRIC_DWARF_ERROR_MAPPING_LOCATE_COUNT,
    METRIC_DWARF_ERROR_MAPPING_LPMTRIE_MISS_COUNT,
    METRIC_DWARF_ERROR_MAPPING_LPMTRIE_NOMAPPING_COUNT,
    METRIC_DWARF_ERROR_MAPPING_LPMTRIE_MALFORMED_COUNT,
    METRIC_DWARF_ERROR_MAPPING_NOBINARYID_COUNT,
    METRIC_DWRAF_ERROR_MAPPING_NOBINARYROOT_COUNT,
    METRIC_DWARF_ERROR_MAPPING_UNWINDTABLELOOKUP_COUNT,
    // --- per-process dwarf metrics ---
    // we found rule in per-process unwind table
    METRIC_DWARF_SUCCESFUL_PER_PROCESS_RULE_LOOKUP_COUNT,
    // process has per-process unwind table, but we did not find rule in there
    // (and also in a per-mapping unwind table)
    METRIC_DWARF_ERROR_PER_PROCESS_RULE_LOOKUP_COUNT,

    METRIC_FP_ERROR_READ_RETURNADDRESS_COUNT,
    METRIC_FP_ERROR_READ_BASEPOINTER_COUNT,
    METRIC_UPROBE_TRIGGERED_COUNT,

    // Lua metrics
    METRIC_ERROR_STAGE_COLLECT_LUA_STACK_COUNT, // Amount of errors during setup stage. See `profiler_stage_collect_lua_stack`
    METRIC_LUA_PROCESSED_STACKS_COUNT, // Amount of stacks processing attempts. We know there is a LuaJIT

    // Lua state searching metrics
    METRIC_LUA_VALID_CACHE_COUNT, // Amount of valid cache usage
    METRIC_LUA_INVALIDED_CACHE_COUNT, // Amount of cache invalidation
    METRIC_LUA_NOT_IN_LUAJIT_BINARY_COUNT, // How many times rip wasn't in LuaJIT binary address range
    METRIC_LUA_GLOBAL_STATE_FOUND_COUNT, // Amount of successful global state findings
    METRIC_LUA_GLOBAL_STATE_NOT_FOUND_COUNT, // Amount of unsuccessful global state findings

    // Lua state validity check metrics
    METRIC_LUA_CUR_L_IS_NULL_COUNT, // Amount of NULLs encountered in `cur_L` in `is_valid_global_state`
    METRIC_LUA_G_EQ_G_MISMATCH_COUNT, // Amount of mismatches of `G(g->cur_L) == g` check
    // METRIC_LUA_L_EQ_L_MISMATCH_COUNT, // Amount of mismatches of `G(g->cur_L)->cur_L == g->cur_L` check. The current difference is that lhs second cur_L is taken from headers, not from binary.
    METRIC_LUA_CUR_L_READ_FAIL_COUNT, // Amount of failed reads of `g->cur_L` in `get_lua_state_from_global_state`

    // Lua stack walk metrics
    METRIC_LUA_NULL_STATE_COUNT, // Amount of NULLs encountered in lua_State / global_State pointers
    METRIC_LUA_PROCESSED_FRAMES_COUNT, // Amount of successfully processed frames in this stack
    METRIC_LUA_GET_FUNCTION_INFO_FAIL_COUNT, // Amount of failed reads of frame functions
    METRIC_LUA_BROKEN_FRAME_COUNT, // Amount of encountered broken frames. Subset of `METRIC_LUA_GET_FUNCTION_INFO_FAIL_COUNT`
    METRIC_LUA_FRAME_IS_NULL_COUNT, // Amount of NULLs encountered in `frame` in `lua_get_function_info`
    METRIC_LUA_FUNCTION_IS_NULL_COUNT, // Amount of NULLs encountered in `fn` in `lua_get_function_info`
    METRIC_LUA_CACHE_MISMATCH_COUNT, // Amount of bpf map symbol cache mismatch. Ideally must be 0

    // Lua common metrics
    METRIC_LUA_DEREF_ERROR_COUNT, // Amount of failed reads of user space memory from a pointer

    // Must be last
    METRIC_COUNT
};

////////////////////////////////////////////////////////////////////////////////

BPF_MAP(metrics, BPF_MAP_TYPE_PERCPU_ARRAY, u32, u64, METRIC_COUNT);

// Make enum metrics available for the userspace.
BTF_EXPORT(enum metric);

////////////////////////////////////////////////////////////////////////////////

static ALWAYS_INLINE void metric_add(enum metric metric, u64 delta) {
    if (!delta) {
        return;
    }

    u64* value = bpf_map_lookup_elem(&metrics, &metric);
    if (!value) {
        return;
    }

    __sync_fetch_and_add(value, delta);
}

static ALWAYS_INLINE void metric_increment(enum metric metric) {
    metric_add(metric, 1);
}

////////////////////////////////////////////////////////////////////////////////
