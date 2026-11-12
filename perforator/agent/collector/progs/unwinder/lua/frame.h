#pragma once

#include "luajit.h"
#include "stack/context.h"
#include "symbol.h"
#include "trace.h"

// namespace lua::frame

/**
 * @brief Small context for current frame.
 */
struct lua_frame_context {
    luajit_tvalue* frame;           // Current frame.
    luajit_tvalue* next_frame;      // Frame above the current.
    luajit_gc_func* frame_function; // Current frame function.
};

/**
 * @brief Checks if this frame already has a saved symbol in the symbol cache.
 *
 * @param proto Proto from lua function in the frame.
 * @param first_line First line where proto is defined.
 * @param cache_key Key for symbol cache.
 *
 * @return `true` if already saved this symbol
 */
static ALWAYS_INLINE bool lua_frame_has_symbol(
    luajit_gc_proto* proto,
    i32 first_line,
    struct interpreter_symbol_key* cache_key
) {
    cache_key->symbol_key = (struct symbol_key) {
        .object_addr = (u64)proto,
        .linestart = first_line,
    };
    return bpf_map_lookup_elem(&interpreter_symbols, cache_key) != NULL;
}

/**
 * @brief Save symbol info for the frame to the map.
 *
 * @param proto Proto from lua function in the frame.
 * @param first_line First line where proto is defined.
 * @param cache_key Key for symbol cache.
 * @param symbol Symbol info.
 */
static ALWAYS_INLINE void lua_frame_save_symbol(
    luajit_gc_proto* proto,
    i32 first_line,
    struct interpreter_symbol_key* cache_key,
    struct symbol* symbol
) {
    cache_key->symbol_key = (struct symbol_key) {
        .object_addr = (u64)proto,
        .linestart = first_line,
    };
    bpf_map_update_elem(&interpreter_symbols, cache_key, symbol, BPF_ANY);
}

/**
 * @brief Check the top vararg frame for correct initialization.
 *
 * Sample might interrupt in-between vararg function which has 2 frames: `FRAME_VARG` and `FRAME_*`.
 * If the top frame is not `FRAME_VARG` but holds a vararg function, it should be skipped.
 *
 * @param frame Top frame.
 * @return `true` is first frame is valid vararg or not a vararg
 */
static ALWAYS_INLINE bool lua_frame_should_skip_vararg(const luajit_tvalue* frame) {
    if (luajit_frame_isvarg(frame)) {
        return true;
    }

    luajit_gc_func* frame_function = (luajit_gc_func*)luajit_frame_func(frame);
    if (frame_function == NULL) {
        LUA_LOG_ERROR("First frame=%px function is NULL", frame);
        return true;
    }

    if (!luajit_isluafunc(frame_function)) {
        return true;
    }

    luajit_gc_proto* proto = luajit_funcproto(frame_function);
    if (proto == NULL) {
        LUA_LOG_ERROR("First frame=%px function=%px proto is NULL", frame, frame_function);
        return false;
    }

    // Function must not have a vararg flag
    return (luajit_gc_proto_get_flags(proto) & LUAJIT_PROTO_VARARG) == 0;
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

    LUA_LOG_DEBUG("Invalid frame error=%d", error);
}

/**
 * @brief Get bytecode position in proto.
 *
 * @param proto Current frame `GCproto`.
 * @param bytecode_pc Pointer to bytecode.
 * @return u32 Bytecode position or `LUAJIT_NO_BCPOS`.
 */
static ALWAYS_INLINE u32 luajit_gc_proto_get_pc_position(luajit_gc_proto* proto, const u32* bytecode_pc) {
    const u32* proto_bc = luajit_gc_proto_bc(proto);
    if (!bytecode_pc || bytecode_pc <= proto_bc) {
        return LUAJIT_NO_BCPOS;
    }

    u64 offset = (u64)bytecode_pc - (u64)proto_bc;
    if (offset % sizeof(u32) || offset > luajit_gc_proto_get_sizebc(proto) * sizeof(u32)) {
        return LUAJIT_NO_BCPOS;
    }

    return luajit_gc_proto_bcpos(proto, bytecode_pc) - 1;
}

/**
 * @brief Return bytecode position for Lua function in the current frame or `LUAJIT_NO_BCPOS`.
 *
 * @see `debug_framepc` from LuaJIT source code.
 *
 * @param frame_context Current frame context.
 * @param proto Current frame `GCproto`.
 * @param state Current `lua_State`.
 * @param bytecode_pc_register Register possibly containing the bytecode PC.
 * @return u32 Last bytecode position in the frame or `LUAJIT_NO_BCPOS`.
 */
static ALWAYS_INLINE u32 lua_frame_get_pc(struct lua_frame_context* frame_context, luajit_gc_proto* proto, luajit_state* lua_state, u32* bytecode_pc_register) {
    const u32* pc;

    const luajit_tvalue* next_frame = frame_context->next_frame;
    if (next_frame == NULL) {
        // When the top frame is a Lua function, it usually doesn't leave ASM VM.
        // If we interruped inside ASM VM, we might have a pointer to current bytecode program counter in designated register.
        u32 pos = luajit_gc_proto_get_pc_position(proto, bytecode_pc_register);
        if (pos != LUAJIT_NO_BCPOS) {
            LUA_LOG_DEBUG("Bytecode pc register=%px was used to determine current pc for frame_function=%px frame=%px", bytecode_pc_register, frame_context->frame_function, frame_context->frame);
            return pos;
        } else {
            LUA_LOG_DEBUG("Bytecode pc register=%px was out of bounds for frame_function=%px frame=%px", bytecode_pc_register, frame_context->frame_function, frame_context->frame);
        }

        luajit_cframe* cf = luajit_cframe_raw(luajit_state_get_cframe(lua_state));
        if (cf == NULL || luajit_cframe_get_pc(cf) == (u32*)luajit_cframe_get_l(cf)) {
            return LUAJIT_NO_BCPOS;
        }

        pc = luajit_cframe_get_pc(cf);
    } else if (luajit_frame_islua(next_frame)) {
        pc = luajit_frame_pc(next_frame);
    } else if (luajit_frame_iscont(next_frame)) {
        pc = luajit_frame_contpc(next_frame);
    } else {
        // Not implemented yet. Lua function below errfunc/gc/hook. Not a hot path.
        LUA_DEV_ASSERT(false, "Lua function below errfunc/gc/hook");
        return LUAJIT_NO_BCPOS;
    }

    // TODO: JIT synthetic PCs.

    return luajit_gc_proto_get_pc_position(proto, pc);
}

/**
 * @brief Get line number for a bytecode position.
 *
 * @see `lj_debug_line` from LuaJIT source code.
 *
 * @param proto Current frame `GCproto`.
 * @param bytecode_pc Bytecode PC.
 * @return Line for given bytecode PC
 */
[[nodiscard]] static ALWAYS_INLINE i32 luajit_gc_proto_get_pc_line(luajit_gc_proto* proto, u32 bytecode_pc) {
    const void* lineinfo = luajit_gc_proto_get_lineinfo(proto);
    u32 sizebc = luajit_gc_proto_get_sizebc(proto);

    if (!lineinfo || bytecode_pc > sizebc) {
        return 0;
    }

    i32 first = luajit_gc_proto_get_firstline(proto);
    i32 numline = luajit_gc_proto_get_numline(proto);

    if (bytecode_pc == sizebc) {
        return first + numline;
    }

    if (bytecode_pc-- == 0) {
        return first;
    }

    if (numline < 256) {
        return first + (i32)BPF_PROBE_READ_USER_FROM(&((u8*)lineinfo)[bytecode_pc]);
    }

    if (numline < 65536) {
        return first + (i32)BPF_PROBE_READ_USER_FROM(&((u16*)lineinfo)[bytecode_pc]);
    }

    return first + (i32)BPF_PROBE_READ_USER_FROM(&((u32*)lineinfo)[bytecode_pc]);
}

/**
 * @brief Set frame as Lua frame.
 *
 * @param stack_context Stack context.
 * @param frame_context Current frame context.
 * @return Currently always `true`.
 */
[[nodiscard]] static ALWAYS_INLINE bool lua_frame_set_lua(
    struct lua_stack_context* stack_context,
    struct lua_frame_context* frame_context
) {
    luajit_gc_proto* proto = luajit_funcproto(frame_context->frame_function);
    i32 first_line = luajit_gc_proto_get_firstline(proto);

    u32 pc = lua_frame_get_pc(
        frame_context, proto,
        (luajit_state*)stack_context->lua_state,
        (u32*)stack_context->bytecode_pc_register
    );
    LUA_LOG_DEBUG("Lua frame function=%px pc=%d", frame_context->frame_function, pc);

    struct lua_frame* lua_frame = &stack_context->lua_frame;
    *lua_frame = (struct lua_frame) {
        .type = LUA_FRAME_TYPE_LUA,
        .value.lua_frame = {
            .proto_address = (u64)proto,
            .first_line = first_line,
            .current_line = pc == LUAJIT_NO_BCPOS ? first_line : luajit_gc_proto_get_pc_line(proto, pc),
            .bytecode_index = pc,
        },
    };
    LUA_LOG_DEBUG("Lua frame function=%px proto=%px current_line=%d", frame_context->frame_function, proto, lua_frame->value.lua_frame.current_line);

    if (lua_frame_has_symbol(proto, first_line, &stack_context->symbol_cache_key)) {
        LUA_LOG_DEBUG("Lua frame symbol cache hit proto=%px", proto);
        return true;
    }

    struct symbol* symbol = &stack_context->symbol;
    char* caret = symbol->data;
    u8 name_length = 0;

    if (!first_line && luajit_gc_proto_get_numline(proto)) {
        name_length = (u8)LUA_SYMBOL_APPEND_LITERAL(symbol->data, "in main chunk");
    }

    symbol->name_length = name_length;
    caret += name_length;

    // This must be written inline for least amount of verifier checks
    const char* filename = luajit_proto_chunknamestr(proto);
    long status = bpf_probe_read_user_str(caret, SYMBOL_BUFFER_SIZE - name_length, filename);
    if (status <= 0) {
        LUA_LOG_ERROR("Failed to read proto=%px filename (%d)", proto, status);
        symbol->filename_length = (u8)lua_symbol_append_fail(caret);
    } else {
        --status;
        symbol->filename_length = status > 255 ? 255 : (u8)status;
    }

    LUA_LOG_DEBUG("Saved symbol for Lua frame proto=%px filename_length=%d", proto, symbol->filename_length);
    lua_frame_save_symbol(proto, first_line, &stack_context->symbol_cache_key, symbol);
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
            .function_address = (u64)luajit_gc_func_get_f(function),
            .ffid = luajit_gc_func_get_ffid(function),
        },
    };

    LUA_LOG_DEBUG("C frame function=%px function_address=%px ffid=%d", function, lua_frame->value.c_frame.function_address, lua_frame->value.c_frame.ffid);
}
