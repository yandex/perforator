#pragma once

#include "../../metrics.h"
#include "../luajit/luajit.h"
#include "../symbol.h"
#include "../trace.h"
#include "../types.h"

#include "pack.h"

// namespace lua::frame

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
[[nodiscard]] static ALWAYS_INLINE cTValue *lua_frame_next(cTValue *frame) {
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
 * @brief Gets current uncommitted frame.
 *
 * @note Increments stack size.
 *
 * @param state Lua unwind state.
 * @return struct interpreter_frame* Current uncommitted frame.
 */
static ALWAYS_INLINE struct interpreter_frame *
lua_frame_new(struct lua_state *state) {
    __auto_type interpreter_frame = &state->stack.frames[state->stack.len++];

    interpreter_frame->symbol_key.pid = state->pid;

    return interpreter_frame;
}

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
 * @return `true` is pushed successfully.
 */
[[nodiscard]] static ALWAYS_INLINE bool
lua_frame_push_lua(struct interpreter_frame *interpreter_frame,
                   struct symbol *symbol, GCfunc *function) {
    GCproto *proto = funcproto(function);

    if (!proto) {
        return false;
    }

    __auto_type object_address = lua_frame_pack_lua_object_address(proto);
    BCLine line_defined = BPF_PROBE_READ_USER(proto, firstline);
    lua_frame_set_key(interpreter_frame, object_address, line_defined);

    lua_symbol_new(symbol);
    char *caret = symbol->data;

    if ((line_defined || !BPF_PROBE_READ_USER(proto, numline))) {
        size_t length = LUA_SYMBOL_APPEND_LITERAL(caret, "<no name>");
        caret += length;
        symbol->name_length = length;
    } else {
        size_t length = LUA_SYMBOL_APPEND_LITERAL(caret, "in main chunk");
        caret += length;
        symbol->name_length = length;
    }

    const char *filename = strdata(proto_chunkname(proto));

    // This must be written inline for least amount of verifier checks
    long status = bpf_probe_read_user_str(
        caret, SYMBOL_BUFFER_SIZE - symbol->name_length, filename);
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

/**
 * @brief Get info about the function on this frame.
 *
 * @param state Lua unwind state.
 * @param frame Current frame.
 * @param frame_gc `frame_gc(frame)`.
 * @return bool
 */
static ALWAYS_INLINE bool lua_frame_get_info(struct lua_state *state,
                                             cTValue *frame, GCobj *frame_gc) {
    __auto_type interpreter_frame = lua_frame_new(state);

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
        metric_increment(METRIC_LUA_BROKEN_FRAME_COUNT);
        return false;
    }

    if (isluafunc(fn)) {
        return lua_frame_push_lua(interpreter_frame, &state->symbol, fn);
    }

    // C and FF functions are handled in the same way
    lua_frame_push_c(interpreter_frame, fn);
    return true;
}
