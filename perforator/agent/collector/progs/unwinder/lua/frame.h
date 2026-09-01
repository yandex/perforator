#pragma once

#include "luajit.h"
#include "symbol.h"
#include "trace.h"

// namespace lua::frame

/**
 * @brief Checks if this frame already has a saved symbol in the symbol cache.
 *
 * @param lua_frame_key Lua frame key.
 *
 * @return `true` if already saved this symbol
 */
static ALWAYS_INLINE bool lua_frame_has_symbol(struct symbol_key* lua_frame_key) {
    return bpf_map_lookup_elem(&interpreter_symbols, lua_frame_key) != NULL;
}

/**
 * @brief Save symbol info for the frame to the map.
 *
 * @param lua_frame_key Lua frame key.
 * @param symbol Symbol info.
 */
static ALWAYS_INLINE void lua_frame_save_symbol(struct symbol_key* lua_frame_key, struct symbol* symbol) {
    bpf_map_update_elem(&interpreter_symbols, lua_frame_key, symbol, BPF_ANY);
}

/**
 * @brief Set frame as invalid frame.
 *
 * @param lua_frame Current frame.
 * @param error Error code.
 */
static ALWAYS_INLINE void lua_frame_set_invalid(struct lua_frame* lua_frame, enum lua_frame_error error) {
    *lua_frame = (struct lua_frame) {
        .type = LUA_FRAME_TYPE_INVALID,
        .value.invalid_frame.error = error,
    };
}

/**
 * @brief Set frame as Lua frame.
 *
 * @param lua_frame Current frame.
 * @param symbol Symbol from state.
 * @param function Lua function.
 * @return `true` if set successfully, `false` if failed to get function proto.
 */
[[nodiscard]] static ALWAYS_INLINE bool lua_frame_set_lua(struct lua_frame* lua_frame, struct symbol* symbol, u32 pid, luajit_gc_func* function) {
    luajit_gc_proto* proto = luajit_funcproto(function);
    i32 line_defined = luajit_gc_proto_get_firstline(proto);

    *lua_frame = (struct lua_frame) {
        .type = LUA_FRAME_TYPE_LUA,
        .value.lua_frame = {
            .object_addr = (u64)proto,
            .pid = pid,
            .linestart = line_defined,
        },
    };

    if (lua_frame_has_symbol(&lua_frame->value.lua_frame)) {
        return true;
    }

    char* caret = symbol->data;
    u8 name_length = 0;

    if (!line_defined && luajit_gc_proto_get_numline(proto)) {
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

    lua_frame_save_symbol(&lua_frame->value.lua_frame, symbol);
    return true;
}

/**
 * @brief Set frame as C/FF frame.
 * FF means FastFunction.
 *
 * @param lua_frame Current frame.
 * @param function C/FF function
 */
static ALWAYS_INLINE void lua_frame_set_c(struct lua_frame* lua_frame, luajit_gc_func* function) {
    *lua_frame = (struct lua_frame) {
        .type = LUA_FRAME_TYPE_C,
        .value.c_frame = {
            .object_addr = (u64)luajit_gc_func_get_f(function),
            .ffid = luajit_gc_func_get_ffid(function),
        },
    };
}
