#pragma once

#include "../frame/frame.h"
#include "../state.h"
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
 * At this stage frame is known to be useful. Function checks what type of frame it is and pushes info in proper format.
 * Stack context will be modified to pass filled information about the `interpreter_frame`.
 *
 * @param context Lua stack context.
 * @param frame Current frame. Not NULL.
 * @param frame_gc `frame_gc(frame)`.
 * @return bool
 */
[[nodiscard]] static ALWAYS_INLINE bool lua_stack_process_frame(struct lua_stack_context* context, const luajit_tvalue* frame, luajit_gc_obj* frame_gc) {
    struct interpreter_frame* interpreter_frame = &context->interpreter_frame;

    luajit_gc_func* fn = (luajit_gc_func*)frame_gc;
    if (!fn) {
        LUA_TRACE("[error] lua_stack_process_frame: invalid frame=%px, GCfunc is NULL.", frame);
        lua_frame_set_invalid(interpreter_frame, LUA_STACK_WALK_ERROR_GCFUNC_IS_NULL, 0);
        return false;
    }

    // Probably it's impossible to cover every possible case to always get valid top frame.
    // Probably the top frame is invalid, but others below are valid. Continue, mark frame as invalid.
    if (!luajit_tvisfunc(frame - LUAJIT_LJ_FR2)) {
        LUA_TRACE("[error] lua_stack_process_frame: invalid frame=%px, frame doesn't contain a function, got %d", frame, ~luajit_itype(frame - LUAJIT_LJ_FR2));
        lua_frame_set_invalid(interpreter_frame, LUA_STACK_WALK_ERROR_FRAME_IS_NOT_FUNC, ~luajit_itype(frame - LUAJIT_LJ_FR2));
        return true;
    }

    if (luajit_isluafunc(fn)) {
        return lua_frame_set_lua(interpreter_frame, &context->symbol, fn);
    }

    // C and FF functions are handled in the same way
    lua_frame_set_c(interpreter_frame, fn);
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
 * @return const luajit_tvalue* Next frame below current.
 */
[[nodiscard]] static ALWAYS_INLINE const luajit_tvalue* lua_stack_next_frame(const luajit_tvalue* frame) {
    if (luajit_frame_islua(frame)) {
        frame = luajit_frame_prevl(frame);

        // Skip vararg pseudo-frame. (if varg: fallthrough)
        if (!luajit_frame_isvarg(frame)) {
            return frame;
        }
    }

    return luajit_frame_prevd(frame);
}

/**
 * @brief Step the stack once.
 *
 * @note This function is made global to reduce verifier passes in `lua_stack_walk`.
 * To communicate with caller function `lua_stack_context` is used.
 *
 * @return enum lua_stack_step_result Result code.
 */
[[nodiscard]] NOINLINE enum lua_stack_step_result lua_stack_step() {
    struct lua_stack_context* context = lua_stack_context_get();
    if (context == NULL) {
        LUA_TRACE("[error] lua_stack_step: failed to get lua_stack_context");
        return LUA_STACK_STEP_RESULT_STOP;
    }

    const luajit_tvalue* frame = (const luajit_tvalue*)context->frame;
    const luajit_tvalue* max_stack = (const luajit_tvalue*)context->max_stack;
    const luajit_tvalue* bottom = (const luajit_tvalue*)context->bottom;

    if (frame <= bottom) {
        return LUA_STACK_STEP_RESULT_STOP;
    }

    if (frame >= max_stack) {
        LUA_TRACE("[error] lua_stack_step: broken frame");
        return LUA_STACK_STEP_RESULT_STOP;
    }

    context->frame = (u64)lua_stack_next_frame(frame);

    luajit_gc_obj* frame_gc = luajit_frame_gc(frame);

    // Skip dummy frames. See lj_err_optype_call().
    bool is_dummy_frame = frame_gc == (luajit_gc_obj*)(context->current_lua_state);
    if (is_dummy_frame) {
        return LUA_STACK_STEP_RESULT_CONTINUE;
    }

    if (!lua_stack_process_frame(context, frame, frame_gc)) {
        metric_increment(METRIC_LUA_PROCESSED_FRAMES_FAIL_COUNT);
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
static ALWAYS_INLINE void lua_stack_reset(struct lua_state* state) {
    state->stack.len = 0;
}

/**
 * @brief Add frame to the stack.
 *
 * @param state Lua unwind state.
 * @param frame Interpreter frame.
 */
static ALWAYS_INLINE void lua_stack_push(struct lua_state* state, struct interpreter_frame interpreter_frame) {
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
static ALWAYS_INLINE void lua_stack_walk(struct lua_state* state) {
    luajit_state* lua_state = lua_state_get_lua_state(state);
    luajit_global_state* global_state = lua_state_get_global_state(state);

    int vmstate = luajit_global_state_get_vm_state(global_state, &state->config);

    // VM is idle
    if (vmstate == ~LUAJIT_VM_STATE_INTERPRETING && !luajit_state_get_cframe(lua_state)) {
        return;
    }

    const luajit_tvalue* base = luajit_state_get_base(lua_state);
    const luajit_tvalue* max_stack = luajit_state_get_maxstack(lua_state);
    const luajit_tvalue* bottom = luajit_state_get_stack(lua_state) + LUAJIT_LJ_FR2;

    if (vmstate == ~LUAJIT_VM_STATE_INTERPRETING) {
        // VM is interpreting
        base = lua_state_resolve_base(state, base, max_stack, bottom);
    } else if (vmstate >= 0) {
        // VM is executing a JIT trace
        base = lua_state_get_jit_base(state, base, global_state);
    }

    const luajit_tvalue* frame = base - 1;

    // This might happen if we stopped before full initialize of vararg function frames.
    if (luajit_frame_isvarg(frame)) {
        frame = luajit_frame_prevd(frame);
    }

    struct lua_stack_context* context = lua_stack_context_get();
    if (context == NULL) {
        LUA_TRACE("[error] lua_stack_walk: failed to get lua_stack_context");
        return;
    }
    lua_stack_context_init(context, state, frame, max_stack, bottom);

    for (int i = 0; i < LUA_MAX_STACK_DEPTH; ++i) {
        enum lua_stack_step_result status = lua_stack_step();

        if (status & LUA_STACK_STEP_RESULT_HAS_FRAME) {
            lua_stack_push(state, context->interpreter_frame);
        }

        if ((status & LUA_STACK_STEP_RESULT_CONTINUE) == 0) {
            return;
        }
    }
}
