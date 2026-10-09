#pragma once

#include <bpf/bpf.h>

#include "../../interpreter/types.h"
#include "../../output.h"

#include "../luajit.h"
#include "../trace.h"
#include "../types.h"

// namespace lua::stack::context

/**
 * @brief Stack context used to communicate between BPF functions.
 */
struct lua_stack_context {
    u64 frame;                                      // Current frame.
    u64 next_frame;                                 // TODO: Next (above current) frame. Was read before `frame`.
    u64 max_stack;                                  // Last free slot in the stack.
    u64 bottom;                                     // Last frame in the stack.
    u64 lua_state;                                  // Current `lua_State*`.
    struct lua_frame lua_frame;                     // Current interpreter frame.
    u64 bytecode_pc_register;                       // Value of register potentially holding the PC pointer for the top Lua frame.
    struct interpreter_symbol_key symbol_cache_key; // Key for Lua frame. TODO
    struct symbol symbol;                           // Temporary buffer for frame information.
};

BPF_MAP(lua_stack_context, BPF_MAP_TYPE_PERCPU_ARRAY, u32, struct lua_stack_context, 1);

/**
 * @brief Get stack context to exchange information between functions.
 *
 * @return struct lua_stack_context* Stack context.
 */
[[nodiscard]] static ALWAYS_INLINE struct lua_stack_context* lua_stack_context_get() {
    u32 zero = 0;
    return bpf_map_lookup_elem(&lua_stack_context, &zero);
}

/**
 * @brief Initialize stack context before walking the stack.
 *
 * @param context Stack context.
 * @param lua_state Current Lua state.
 * @param frame Current frame.
 * @param max_stack Last free slot in the stack.
 * @param bottom Last frame in the stack.
 * @param bytecode_pc_register Potential PC pointer for the top Lua frame.
 * @param pid Current PID.
 */
static ALWAYS_INLINE void lua_stack_context_init(
    struct lua_stack_context* context, const struct lua_state* state,
    const luajit_tvalue* frame, const luajit_tvalue* max_stack,
    const luajit_tvalue* bottom, u64 bytecode_pc_register
) {
    context->frame = (u64)frame;
    context->next_frame = (u64)NULL;
    context->max_stack = (u64)max_stack;
    context->bottom = (u64)bottom;
    context->lua_state = state->current_lua_state;

    context->lua_frame = (struct lua_frame) {
        .type = LUA_FRAME_TYPE_C,
        .value.c_frame = {
            .function_address = (u64)NULL,
            .ffid = 0,
        },
    };

    context->bytecode_pc_register = bytecode_pc_register;
    context->symbol_cache_key.process_starttime = state->process_starttime;
    context->symbol_cache_key.pid = state->pid;
    context->symbol_cache_key.language = LANGUAGE_LUA;
    ZERO(context->symbol_cache_key._pad);
    context->symbol.codepoint_size = 1; // Lua strings are always utf-8
}

/**
 * @brief Get previous (going top to bottom) frame.
 *
 * Vararg Lua functions has a special treatment because they take 2 frames.
 * Both frames will contain the same Lua function but upper frame has varg type and lower the type of frame before.
 *
 * 0xFF L->base
 * 0xF7 frame type varg (`L->base - 1`)
 * 0xEF varg Lua function
 * ...  varg arguments
 * 0xCF frame type X (depends on the frame below)
 * 0xC7 varg Lua function
 *
 * To save iterations, skip vararg here, take next frame right away.
 *
 * @see `lj_debug_frame` from LuaJIT source code.
 *
 * @param context Stack context.
 */
static ALWAYS_INLINE void lua_stack_context_get_previous_frame(struct lua_stack_context* context) {
    const luajit_tvalue* frame = (luajit_tvalue*)context->frame;

    // **Previous** frame is pseudo-frame, but type of the **current** frame is `FRAME_VARG`.
    if (luajit_frame_isvarg(frame)) {
        LUA_LOG_DEBUG("Skipping pseudo-frame");
        frame = luajit_frame_prevd(frame);
    }

    context->next_frame = (u64)frame;
    context->frame = (u64)luajit_frame_prev(frame);
}
