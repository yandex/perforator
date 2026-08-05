#pragma once

#include "../luajit.h"
#include "../symbol.h"
#include "../trace.h"

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
static ALWAYS_INLINE void lua_frame_set_key(struct interpreter_frame* interpreter_frame, u64 object_address, i32 linestart) {
    struct symbol_key* key = &interpreter_frame->symbol_key;

    key->object_addr = object_address;
    key->linestart = linestart;
    interpreter_frame->position_info = 0;
}

/**
 * @brief Checks if this frame already has a saved symbol in the symbol cache.
 *
 * @param interpreter_frame Current frame.
 *
 * @return `true` if already saved this symbol
 */
static ALWAYS_INLINE bool lua_frame_has_symbol(struct interpreter_frame* interpreter_frame) {
    return bpf_map_lookup_elem(&interpreter_symbols, interpreter_frame) != NULL;
}

/**
 * @brief Save symbol info for the current frame to the map.
 *
 * @param interpreter_frame Current frame.
 * @param symbol Symbol info.
 */
static ALWAYS_INLINE void lua_frame_save_symbol(struct interpreter_frame* interpreter_frame, struct symbol* symbol) {
    bpf_map_update_elem(&interpreter_symbols, interpreter_frame, symbol, BPF_ANY);
}

/**
 * @brief Set frame as invalid frame.
 *
 * @param interpreter_frame Current frame.
 * @param error Error code.
 * @param gct GCobj type found instead of function. Relevant only with `LUA_STACK_WALK_ERROR_FRAME_IS_NOT_FUNC`.
 */
static ALWAYS_INLINE void lua_frame_set_invalid(struct interpreter_frame* interpreter_frame, enum lua_stack_walk_error error, u8 gct) {
    u64 address = lua_frame_pack_invalid_object_address(error, gct);

    lua_frame_set_key(interpreter_frame, address, LUA_LINESTART_NON_LUA_FRAME);
}

/**
 * @brief Set frame as Lua frame.
 *
 * @param interpreter_frame Current frame.
 * @param symbol Symbol from state.
 * @param function Lua function.
 * @return `true` if pushed successfully.
 */
[[nodiscard]] static ALWAYS_INLINE bool lua_frame_set_lua(struct interpreter_frame* interpreter_frame, struct symbol* symbol, luajit_gc_func* function) {
    luajit_gc_proto* proto = luajit_funcproto(function);

    u64 object_address = lua_frame_pack_lua_object_address(proto);
    i32 line_defined = luajit_gc_proto_get_firstline(proto);
    lua_frame_set_key(interpreter_frame, object_address, line_defined);

    if (lua_frame_has_symbol(interpreter_frame)) {
        return true;
    }

    char* caret = symbol->data;
    u8 name_length = 0;

    if (line_defined || !luajit_gc_proto_get_numline(proto)) {
        name_length = (u8)LUA_SYMBOL_APPEND_LITERAL(symbol->data, "<no name>");
    } else {
        name_length = (u8)LUA_SYMBOL_APPEND_LITERAL(symbol->data, "in main chunk");
    }

    symbol->name_length = name_length;
    caret += name_length;

    // This must be written inline for least amount of verifier checks
    const char* filename = luajit_proto_chunknamestr(proto);
    long status = bpf_probe_read_user_str(caret, SYMBOL_BUFFER_SIZE - name_length, filename);
    if (status <= 0) {
        LUA_TRACE("[error] lua_frame_set_lua: failed to read proto=%px filename (%d)", proto, status);
        symbol->filename_length = (u8)lua_symbol_append_fail(caret);
    } else {
        --status;
        symbol->filename_length = status > 255 ? 255 : (u8)status;
    }

    lua_frame_save_symbol(interpreter_frame, symbol);
    return true;
}

/**
 * @brief Set frame as C/FF frame.
 * FF means FastFunction.
 *
 * @param interpreter_frame Current frame.
 * @param function C/FF function
 */
static ALWAYS_INLINE void lua_frame_set_c(struct interpreter_frame* interpreter_frame, luajit_gc_func* function) {
    u64 object_address = lua_frame_pack_c_object_address(luajit_gc_func_get_f(function), luajit_gc_func_get_ffid(function));

    lua_frame_set_key(interpreter_frame, object_address, LUA_LINESTART_NON_LUA_FRAME);
}
