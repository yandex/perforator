#pragma once

#include "../frame/frame.h"
#include "../state/state.h"
#include "../types.h"
#include "context.h"

// namespace lua::stack

/**
 * @brief `lua_stack_step` function flags.
 */
enum lua_stack_step_result {
    LUA_STACK_STEP_RESULT_STOP = 0,          // Stop the loop
    LUA_STACK_STEP_RESULT_CONTINUE = 1 << 0, // Continue the loop
    LUA_STACK_STEP_RESULT_HAS_FRAME = 1 << 1 // New frame was made
};

/**
 * @brief Process valid frame.
 *
 * At this stage frame is known to be useful. Function checks what type of frame
 * it is and pushes info in proper format.
 * Stack context will be modified to pass filled information about the
 * `interpreter_frame`.
 *
 * @param context Lua stack context.
 * @param frame Current frame.
 * @param frame_gc `frame_gc(frame)`.
 * @return bool
 */
[[nodiscard]] static ALWAYS_INLINE bool
lua_stack_process_frame(struct lua_stack_context *context, cTValue *frame,
                        GCobj *frame_gc) {
    __auto_type interpreter_frame = &context->interpreter_frame;

    if (!frame) {
        lua_frame_push_invalid(interpreter_frame,
                               LUA_STACK_WALK_ERROR_FRAME_IS_NULL, 0);
        metric_increment(METRIC_LUA_FRAME_IS_NULL_COUNT);
        return false;
    }

    GCfunc *fn = gco2func(frame_gc);
    if (!fn) {
        lua_frame_push_invalid(interpreter_frame,
                               LUA_STACK_WALK_ERROR_GCFUNC_IS_NULL, 0);
        metric_increment(METRIC_LUA_FUNCTION_IS_NULL_COUNT);
        return false;
    }

    if (!tvisfunc(frame - LJ_FR2)) {
        // TODO: Is it because of some failed read before this check?
        lua_frame_push_invalid(interpreter_frame,
                               LUA_STACK_WALK_ERROR_FRAME_IS_NOT_FUNC,
                               ~itype(frame - LJ_FR2));
        metric_increment(METRIC_LUA_FRAME_IS_NOT_FUNC_COUNT);
        return false;
    }

    if (isluafunc(fn)) {
        return lua_frame_push_lua(interpreter_frame, &context->symbol, fn);
    }

    // C and FF functions are handled in the same way
    lua_frame_push_c(interpreter_frame, fn);
    return true;
}

/**
 * @brief Get next (going top to bottom) frame.
 *
 * Vararg Lua functions has a special treatment because they take 2 frames.
 * First goes Lua frame, then its vararg.
 * To save iterations, skip vararg here, take next frame right away.
 *
 * @see `lj_debug_frame` from LuaJIT source code.
 *
 * @param frame Current frame.
 * @return cTValue* Next frame below current.
 */
[[nodiscard]] static ALWAYS_INLINE cTValue *
lua_stack_next_frame(cTValue *frame) {
    if (frame_islua(frame)) {
        return frame_prevl(frame);
    }

    if (frame_isvarg(frame)) {
        // Skip vararg pseudo-frame.
        return frame_prev(frame_prevd(frame));
    }

    return frame_prevd(frame);
}

/**
 * @brief Step the stack once.
 *
 * @note This function is made global to reduce verifier passes in
 * `lua_stack_walk`.
 * To communicate with caller function `lua_stack_context` is
 * used.
 *
 * @return enum lua_stack_step_result Result code.
 */
[[nodiscard]] NOINLINE enum lua_stack_step_result lua_stack_step() {
    struct lua_stack_context *context = lua_stack_context_get();
    if (context == NULL) {
        LUA_TRACE("lua_stack_step failed to get lua_stack_context");
        return LUA_STACK_STEP_RESULT_STOP;
    }

    __auto_type frame = (cTValue *)context->frame;
    __auto_type max_stack = (cTValue *)context->max_stack;
    __auto_type bottom = (cTValue *)context->bottom;

    if (frame <= bottom) {
        return LUA_STACK_STEP_RESULT_STOP;
    }

    if (frame >= max_stack) {
        metric_increment(METRIC_LUA_BROKEN_FRAME_COUNT);
        return LUA_STACK_STEP_RESULT_STOP;
    }

    context->frame = (u64)lua_stack_next_frame(frame);

    __auto_type frame_gc = frame_gc(frame);
    bool is_dummy_frame = frame_gc == obj2gco(context->L);

    // Skip dummy frames. See lj_err_optype_call().
    if (is_dummy_frame) {
        return LUA_STACK_STEP_RESULT_CONTINUE;
    }

    if (!lua_stack_process_frame(context, frame, frame_gc)) {
        metric_increment(METRIC_LUA_FRAME_GET_INFO_FAIL_COUNT);
        return LUA_STACK_STEP_RESULT_STOP | LUA_STACK_STEP_RESULT_HAS_FRAME;
    }

    metric_increment(METRIC_LUA_PROCESSED_FRAMES_COUNT);
    return LUA_STACK_STEP_RESULT_CONTINUE | LUA_STACK_STEP_RESULT_HAS_FRAME;
}

/**
 * @brief Reset the stack to empty state.
 *
 * @param state Lua unwind state.
 */
static ALWAYS_INLINE void lua_stack_reset(struct lua_state *state) {
    state->stack.len = 0;
}

/**
 * @brief Add frame to the stack.
 *
 * @param state Lua unwind state.
 * @param frame Interpreter frame.
 */
static ALWAYS_INLINE void
lua_stack_push(struct lua_state *state,
               struct interpreter_frame interpreter_frame) {
    state->stack.len &= LUA_MAX_STACK_DEPTH_VERIFIER_MASK;
    state->stack.frames[state->stack.len] = interpreter_frame;
    ++state->stack.len;
}

/**
 * @brief Main function that walks Lua stack.
 *
 * `L` and `G` are valid.
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
    cTValue *max_stack = tvref(BPF_PROBE_READ_USER(L, maxstack));
    cTValue *bottom = tvref(BPF_PROBE_READ_USER(L, stack)) + LJ_FR2;

    if (vmstate == ~LJ_VMST_INTERP) {
        base = lua_state_resolve_base(state, base, max_stack, bottom);
    } else if (vmstate >= 0) {
        base = lua_state_get_jit_base(state, base, g);
    }

    cTValue *frame = base - 1;

    struct lua_stack_context *context = lua_stack_context_get();
    if (context == NULL) {
        LUA_TRACE("lua_stack_walk failed to get lua_stack_context");
        return;
    }

    lua_stack_context_init(context, state, frame, max_stack, bottom);

    for (int i = 0; i < LUA_MAX_STACK_DEPTH; ++i) {
        __auto_type status = lua_stack_step();

        if (status & LUA_STACK_STEP_RESULT_HAS_FRAME) {
            lua_stack_push(state, context->interpreter_frame);
        }

        if ((status & LUA_STACK_STEP_RESULT_CONTINUE) == 0) {
            return;
        }
    }
}
