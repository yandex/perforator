#pragma once

#include "../frame/frame.h"
#include "../state/state.h"
#include "../types.h"

// namespace lua::stack

/**
 * @brief Resets the stack to empty state.
 *
 * @param state Lua unwind state.
 */
static ALWAYS_INLINE void lua_stack_reset(struct lua_state *state) {
    state->stack.len = 0;
}

/**
 * @brief Main function that walks Lua stack.
 *
 * `L` and `G` are valid.
 * Call stack level: 4
 *
 * @param state Lua unwind state.
 */
static ALWAYS_INLINE void lua_stack_walk(struct lua_state *state) {
    lua_stack_reset(state);

    lua_State *L = lua_state_get_lua_state(state);
    global_State *g = lua_state_get_global_state(state);

// `vmstate` is volatile
#pragma clang diagnostic push
#pragma clang diagnostic ignored                                               \
    "-Wincompatible-pointer-types-discards-qualifiers"
    int vmstate = BPF_PROBE_READ_USER(g, vmstate);
#pragma clang diagnostic pop

    // VM is idle.
    if (vmstate == ~LJ_VMST_INTERP && !BPF_PROBE_READ_USER(L, cframe)) {
        return;
    }

    cTValue *base = BPF_PROBE_READ_USER(L, base);
    cTValue *bottom = tvref(BPF_PROBE_READ_USER(L, stack)) + LJ_FR2;
    cTValue *max_stack = tvref(BPF_PROBE_READ_USER(L, maxstack));

    if (vmstate == ~LJ_VMST_INTERP) {
        base = lua_state_resolve_base(state, base, max_stack, bottom);
    } else if (vmstate >= 0) {
        base = lua_state_get_jit_base(state, base, g);
    }

    cTValue *frame = base - 1;

    for (int i = 0; i < LUA_MAX_STACK_DEPTH && frame > bottom; i++) {
        if (frame >= max_stack) {
            metric_increment(METRIC_LUA_BROKEN_FRAME_COUNT);
            return;
        }

        __auto_type frame_gc = frame_gc(frame);
        bool is_dummy_frame = frame_gc == obj2gco(L);

        // Skip dummy frames. See lj_err_optype_call().
        if (!is_dummy_frame) {
            if (!lua_frame_get_info(state, frame, frame_gc)) {
                metric_increment(METRIC_LUA_FRAME_GET_INFO_FAIL_COUNT);
                return;
            }

            metric_increment(METRIC_LUA_PROCESSED_FRAMES_COUNT);
        }

        frame = lua_frame_next(frame);
    }
}
