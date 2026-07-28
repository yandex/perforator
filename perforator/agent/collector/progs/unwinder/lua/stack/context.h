#pragma once

#include <bpf/bpf.h>

#include "../../interpreter/types.h"
#include "../luajit/luajit.h"
#include "../state.h"

// namespace lua::stack::context

/**
 * @brief Stack context used to communicate between BPF functions.
 */
struct lua_stack_context {
    u64 frame;                                   // Current frame.
    u64 max_stack;                               // Last free slot in the stack.
    u64 bottom;                                  // Last frame in the stack.
    u64 current_lua_state;                       // Current `lua_State*`.
    struct interpreter_frame interpreter_frame;  // Current interpreter frame.
    struct symbol symbol;  // Temporary buffer for frame information.
};

BPF_MAP(lua_stack_context, BPF_MAP_TYPE_PERCPU_ARRAY, u32,
        struct lua_stack_context, 1);

/**
 * @brief Get stack context to exchange information between functions.
 *
 * @return struct lua_stack_context* Stack context.
 */
[[nodiscard]] static ALWAYS_INLINE struct lua_stack_context*
lua_stack_context_get() {
    u32 zero = 0;
    return bpf_map_lookup_elem(&lua_stack_context, &zero);
}

/**
 * @brief Initialize stack context before walking the stack.
 *
 * @param context Stack context.
 * @param state Lua unwind state.
 * @param frame Current frame.
 * @param max_stack Last free slot in the stack.
 * @param bottom Last frame in the stack.
 */
static ALWAYS_INLINE void lua_stack_context_init(
    struct lua_stack_context* context, const struct lua_state* state,
    cTValue* frame, cTValue* max_stack, cTValue* bottom) {
    context->frame = (u64)frame;
    context->max_stack = (u64)max_stack;
    context->bottom = (u64)bottom;
    context->current_lua_state = state->current_lua_state;
    context->interpreter_frame.symbol_key.pid = state->pid;
    context->symbol.codepoint_size = 1;  // Lua strings are always utf-8
}
