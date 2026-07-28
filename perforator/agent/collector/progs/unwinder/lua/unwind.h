#pragma once

_Static_assert(__x86_64__,
               "We currently support x64 only"); // TODO

#include "../metrics.h"
#include "../process.h"
#include "stack/stack.h"
#include "state/state.h"
#include "symbol.h"
#include "types.h"

/**
 * @brief Entry point for collecting stack info from Lua VM.
 *
 * @param process_info Information about the current process, and specifically
 * the binary, where LuaJIT is, located.
 * @param state Lua unwind state. Not to be confused with `lua_State`.
 */
static ALWAYS_INLINE void
lua_collect_stack(const struct process_info *process_info,
                  struct lua_state *state) {
    lua_stack_reset(state);

    if (!is_mapped(process_info->lua_binary)) {
        return;
    }

    if (!lua_state_get_config_from_process(state, process_info)) {
        return;
    }

    metric_increment(METRIC_LUA_PROCESSED_STACKS_COUNT);

    _Static_assert(offsetof(lua_State, base) == 32, "");
    _Static_assert(offsetof(lua_State, top) == 40, "");
    _Static_assert(offsetof(lua_State, maxstack) == 48, "");
    _Static_assert(offsetof(lua_State, stack) == 56, "");
    _Static_assert(offsetof(lua_State, cframe) == 80, "");
    _Static_assert(offsetof(lua_State, stacksize) == 88, "");
    _Static_assert(offsetof(global_State, vmstate) == 184, "");

    if (!lua_state_find_g_and_l(state)) {
        return;
    }

    lua_stack_walk(state);
}
