#pragma once

#include "types.h"

#ifdef __x86_64__

#include "../metrics.h"
#include "../process.h"

#include "stack/stack.h"
#include "state.h"

/**
 * @brief Entry point for collecting stack info from Lua VM.
 * Publishes VM_INFO followed by at least one Lua/C/invalid frame, or an empty stack.
 *
 * @param process_info Information about the current process, and specifically the binary, where LuaJIT is, located.
 * @param state Lua unwind state. Not to be confused with `lua_State`.
 */
static ALWAYS_INLINE void lua_collect_stack(const struct process_info* process_info, struct lua_state* state) {
    lua_stack_reset(state);

    if (!is_mapped(process_info->lua_binary)) {
        return;
    }
    if (!lua_state_get_config_from_process(state, process_info)) {
        return;
    }

    metric_increment(METRIC_LUA_PROCESSED_STACKS_COUNT);

    if (!lua_state_find_g_and_l(state)) {
        return;
    }

    LUA_LOG_DEBUG(">>> Stack processing start");

    // Reserve the first wire frame without publishing a header-only stack.
    state->stack.len = 1;
    lua_stack_walk(state);

    if (state->stack.len == 1) {
        state->stack.len = 0;
        return;
    }

    state->stack.frames[0] = (struct lua_frame) {
        .type = LUA_FRAME_TYPE_VM_INFO,
        .value.vm_info = {
            .vm_start = state->binary_start_address + state->config.vm_start_pc,
            .vm_end = state->binary_start_address + state->config.vm_end_pc,
            .flags = state->stack_flags,
        },
    };

    // Report only interpreter frames; packing adds the leading metadata frame.
    --state->stack.len;
    LUA_LOG_DEBUG("<<< Stack processing end");
}

#endif
