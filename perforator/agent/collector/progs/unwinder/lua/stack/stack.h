#pragma once

#include "../../metrics.h"

#include "../frame.h"
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
 * Stack context will be modified to pass filled information about the `lua_frame`.
 *
 * @param stack_context Stack context.
 * @param frame_context Current frame context.
 * @return `true` if frame was processed successfully, `false` on failure.
 */
[[nodiscard]] static ALWAYS_INLINE bool lua_stack_process_frame(struct lua_stack_context* stack_context, struct lua_frame_context* frame_context) {
    struct lua_frame* lua_frame = &stack_context->lua_frame;

    luajit_gc_func* frame_function = frame_context->frame_function;
    if (!frame_function) {
        LUA_LOG_ERROR("Invalid frame=%px, GCfunc is NULL.", frame_context->frame);
        lua_frame_set_invalid(lua_frame, LUA_FRAME_ERROR_GCFUNC_IS_NULL);
        return false;
    }

    // Probably it's impossible to cover every possible case to always get a valid top frame.
    // Probably the top frame is invalid, but others below are valid. Continue, mark frame as invalid.
    if (!luajit_tvisfunc(frame_context->frame - LUAJIT_LJ_FR2)) {
        LUA_LOG_ERROR("Invalid frame=%px, frame doesn't contain a function, got %d", frame_context->frame, ~luajit_itype(frame_context->frame - LUAJIT_LJ_FR2));
        lua_frame_set_invalid(lua_frame, LUA_FRAME_ERROR_GCFUNC_WRONG_TYPE);
        return true;
    }

    LUA_LOG_DEBUG("Processing frame=%px", frame_function);

    if (luajit_isluafunc(frame_function)) {
        return lua_frame_set_lua(stack_context, frame_context);
    }

    // C and FF functions are handled in the same way
    lua_frame_set_c(lua_frame, frame_function);
    return true;
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
    struct lua_stack_context* stack_context = lua_stack_context_get();
    if (stack_context == NULL) {
        LUA_LOG_ERROR("Failed to get lua_stack_context");
        return LUA_STACK_STEP_RESULT_STOP;
    }

    luajit_tvalue* frame = (luajit_tvalue*)stack_context->frame;
    luajit_tvalue* next_frame = (luajit_tvalue*)stack_context->next_frame;
    luajit_tvalue* max_stack = (luajit_tvalue*)stack_context->max_stack;
    luajit_tvalue* bottom = (luajit_tvalue*)stack_context->bottom;

    // Including NULL
    if (frame <= bottom) {
        LUA_LOG_DEBUG("Frame reached bottom of stack eq=%d", frame == bottom);
        return LUA_STACK_STEP_RESULT_STOP;
    }
    if (frame >= max_stack) {
        LUA_LOG_ERROR("Broken frame=%px (max_stack=%px)", frame, max_stack);
        return LUA_STACK_STEP_RESULT_STOP;
    }

    // Before leaving the function, update context for next iteration.
    // Current frame becomes next, previous becomes current.
    // From this point on, use local variables instead of `context`.
    lua_stack_context_get_previous_frame(stack_context);

    luajit_gc_func* frame_function = luajit_frame_func(frame);

    // Skip dummy frames. See lj_err_optype_call().
    if (frame_function == (void*)(stack_context->lua_state)) {
        LUA_LOG_DEBUG("Skipping dummy frame=%px", frame_function);
        return LUA_STACK_STEP_RESULT_CONTINUE;
    }

    struct lua_frame_context frame_context = {
        .frame = frame,
        .next_frame = next_frame,
        .frame_function = frame_function
    };

    if (!lua_stack_process_frame(stack_context, &frame_context)) {
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
static NOINLINE void lua_stack_push(struct lua_state* state, struct lua_frame* lua_frame) {
    state->stack.len &= LUA_MAX_STACK_DEPTH_VERIFIER_MASK;
    state->stack.frames[state->stack.len] = *lua_frame;
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
        LUA_LOG_DEBUG("VM is idle");
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

    if (frame <= bottom || frame >= max_stack) {
        LUA_LOG_ERROR("Broken frame bottom=%px max_stack=%px frame=%px", bottom, max_stack, frame);
        return;
    }

    struct lua_stack_context* stack_context = lua_stack_context_get();
    if (stack_context == NULL) {
        LUA_LOG_ERROR("Failed to get lua_stack_context");
        return;
    }

    u64 binary_relative_address = state->instruction_pointer - state->binary_start_address;
    bool is_in_luajit_vm = state->config.vm_start_pc <= binary_relative_address && binary_relative_address < state->config.vm_end_pc;
    u64 bytecode_pc = is_in_luajit_vm ? state->bytecode_pc_register : 0;

    lua_stack_context_init(stack_context, state, frame, max_stack, bottom, bytecode_pc);

    for (int i = 0; i < LUA_MAX_STACK_DEPTH; ++i) {
        enum lua_stack_step_result status = lua_stack_step();

        if (status & LUA_STACK_STEP_RESULT_HAS_FRAME) {
            lua_stack_push(state, &stack_context->lua_frame);
        }

        if (!(status & LUA_STACK_STEP_RESULT_CONTINUE)) {
            LUA_LOG_DEBUG("Stopped at frame %d. Frames: %d", i, state->stack.len);
            return;
        }
    }

    LUA_LOG_INFO("Loop reached max stack depth. Frames: %d", state->stack.len);
}
