#pragma once

#include "../../metrics.h"
#include "../luajit/luajit.h"
#include "../symbol.h"
#include "../trace.h"
#include "../types.h"

#include "pack.h"

// namespace lua::frame

/**
 * @brief Set key for the current frame.
 *
 * Key is used to retrieve its name and filename from the symbol cache.
 *
 * @param interpreter_frame Current frame.
 * @param object_address Object address. Must be packed to u64 before this call.
 * @param linestart Line information.
 */
static ALWAYS_INLINE void
lua_frame_set_key(struct interpreter_frame *interpreter_frame,
                  u64 object_address, i32 linestart) {
    __auto_type key = &interpreter_frame->symbol_key;

    key->object_addr = object_address;
    key->linestart = linestart;
}

/**
 * @brief Finishes frame by pushing symbol info to the map and incrementing
 * stack size.
 *
 * @param interpreter_frame Current frame.
 */
static ALWAYS_INLINE void
lua_frame_save_symbol(struct interpreter_frame *interpreter_frame,
                      struct symbol *symbol) {
    bpf_map_update_elem(&interpreter_symbols, interpreter_frame, symbol,
                        BPF_ANY);
}

/**
 * @brief Push invalid frame to stack.
 *
 * @param interpreter_frame Current frame.
 * @param error Error code.
 * @param gct GCobj type found instead of function. Relevant only with
 * `LUA_STACK_WALK_ERROR_FRAME_IS_NOT_FUNC`.
 */
static ALWAYS_INLINE void
lua_frame_push_invalid(struct interpreter_frame *interpreter_frame,
                       enum lua_stack_walk_error error, uint8_t gct) {
    __auto_type addr = lua_frame_pack_invalid_object_address(error, gct);

    lua_frame_set_key(interpreter_frame, addr, LUA_LINESTART_NON_LUA_FRAME);
}

/**
 * @brief Push Lua frame to stack.
 *
 * @param interpreter_frame Current frame.
 * @param symbol Symbol from state.
 * @param function Lua function.
 * @return `true` if pushed successfully.
 */
[[nodiscard]] static ALWAYS_INLINE bool
lua_frame_push_lua(struct interpreter_frame *interpreter_frame,
                   struct symbol *symbol, GCfunc *function) {
    GCproto *proto = funcproto(function);
    if (!proto) {
        lua_frame_push_invalid(interpreter_frame,
                               LUA_STACK_WALK_ERROR_PROTO_IS_NULL, 0);
        metric_increment(METRIC_LUA_PROTO_IS_NULL_COUNT);
        return false;
    }

    __auto_type object_address = lua_frame_pack_lua_object_address(proto);
    BCLine line_defined = BPF_PROBE_READ_USER(proto, firstline);
    lua_frame_set_key(interpreter_frame, object_address, line_defined);

    char *caret = symbol->data;
    size_t name_length = 0;

    if ((line_defined || !BPF_PROBE_READ_USER(proto, numline))) {
        name_length = LUA_SYMBOL_APPEND_LITERAL(symbol->data, "<no name>");
    } else {
        name_length = LUA_SYMBOL_APPEND_LITERAL(symbol->data, "in main chunk");
    }

    symbol->name_length = name_length;
    caret += name_length;

    const char *filename = strdata(proto_chunkname(proto));

    // This must be written inline for least amount of verifier checks
    long status = bpf_probe_read_user_str(
        caret, SYMBOL_BUFFER_SIZE - name_length, filename);
    if (status <= 0) {
        symbol->filename_length = lua_symbol_append_fail(caret);
    } else {
        symbol->filename_length = status - 1;
    }

    lua_frame_save_symbol(interpreter_frame, symbol);
    return true;
}

/**
 * @brief Push C/FF frame to stack.
 *
 * @param interpreter_frame Current frame.
 * @param function C/FF function
 */
static ALWAYS_INLINE void
lua_frame_push_c(struct interpreter_frame *interpreter_frame,
                 GCfunc *function) {
    __auto_type object_address =
        lua_frame_pack_c_object_address(BPF_PROBE_READ_USER(function, c.f),
                                        BPF_PROBE_READ_USER(function, c.ffid));

    lua_frame_set_key(interpreter_frame, object_address,
                      LUA_LINESTART_NON_LUA_FRAME);
}
