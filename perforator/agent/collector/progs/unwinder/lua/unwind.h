#pragma once

#include "stack/walk_error.h"
#include "state.h"
#include "types.h"

#ifdef __x86_64__

#include "../metrics.h"
#include "../process.h"
#include "stack/stack.h"

/**
 * @brief Entry point for collecting stack info from Lua VM.
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

    lua_stack_walk(state);
}

#endif
