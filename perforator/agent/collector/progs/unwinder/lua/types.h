#pragma once

#include "../output.h"

#include "luajit/luajit.h"

/* Read ULEB128 from buffer. */
// BPF version with loop unrolled and reading process memory via BPF API
bool ALWAYS_INLINE LJ_FASTCALL lj_buf_ruleb128(const char **pp, uint32_t *out) {
    const uint8_t *w = (const uint8_t *)*pp;
    uint32_t value;
    uint8_t byte;
    int err = bpf_probe_read_user(&byte, sizeof(byte), w);
    if (err != 0) {
        return false;
    }
    w += 1;
    value = byte;
    if (value < 0x80) {
        *pp = (const char *)w;
        *out = value;
        return true;
    }
    value &= 0x7F;
    uint8_t more = 1;
    int shift = 7;
    // byte 2
    if (more) {
        int err = bpf_probe_read_user(&byte, sizeof(byte), w);
        if (err != 0) {
            return false;
        }
        w += 1;
        value |= (byte & 0x7F) << shift;
        more = (byte & 0x80) ? 1 : 0;
        shift += 7;
    }
    // byte 3
    if (more) {
        int err = bpf_probe_read_user(&byte, sizeof(byte), w);
        if (err != 0) {
            return false;
        }
        w += 1;
        value |= (byte & 0x7F) << shift;
        more = (byte & 0x80) ? 1 : 0;
        shift += 7;
    }
    // byte 4
    if (more) {
        int err = bpf_probe_read_user(&byte, sizeof(byte), w);
        if (err != 0) {
            return false;
        }
        w += 1;
        value |= (byte & 0x7F) << shift;
        more = (byte & 0x80) ? 1 : 0;
        shift += 7;
    }
    // byte 5 (final)
    if (more) {
        int err = bpf_probe_read_user(&byte, sizeof(byte), w);
        if (err != 0) {
            return false;
        }
        w += 1;
        value |= (byte & 0x7F) << shift;
    }
    *pp = (const char *)w;
    *out = value;
    return true;
}

enum {
    LUA_MAX_STACK_DEPTH = ARRAY_SIZE(((struct interpreter_stack *)0)->frames),
    MAX_LJ_BC_MODE = 154,
};

// Keep in sync with frame_decoding_errors @
// perforator/agent/collector/pkg/profiler/stack_processor.go
enum lua_unwind_error : u8 {
    LUA_UNWIND_ERROR_FRAME_IS_NULL,
    LUA_UNWIND_ERROR_GCFUNC_IS_NULL,
    LUA_UNWIND_ERROR_FRAME_IS_NOT_FUNC
};

/**
 * @brief
 *
 */
enum lua_context_type : u8 {
    LUA_CONTEXT_UNKNOWN,
    LUA_CONTEXT_METAMETHOD,
    LUA_CONTEXT_LOCAL,
    LUA_CONTEXT_GLOBAL,
    LUA_CONTEXT_METHOD,
    LUA_CONTEXT_FIELD,
    LUA_CONTEXT_UPVALUE,
};

/**
 * @brief Tagged pointer union for lua frames
 * In canonical form in most architectures user space addresses are 48 bits
 * long.
 * This allows to store additional data in upper 16 bits.
 *
 *               |-------------MSW--------------.--------------LSW-------------|
 * Raw           |-----------------------------u64-----------------------------|
 *
 *               |64-51|---50-48---|-------------------47-0--------------------|
 * Lua frame     |0...0|---ctx_t---|-------------------proto-------------------|
 *
 *               |64-56|---55-48---|-------------------47-0--------------------|
 * C or FF frame |0...0|---ffid----|-------------------addr--------------------|
 *
 *               |64-54|53-50|49-48|-------------------47-0--------------------|
 * Invalid frame |0...0|-gct-|-err-|0.........................................0|
 */
union lua_symbol_key_data {
    struct {
        u64 ptr : 48; // User-space pointer, not necessarily aligned as function
                      // pointer. Takes 0-47 bits.
        union {
            struct {
                // Origin of the call like global/local/upvalue.
                enum lua_context_type context_type : 3;
            } lua_frame_data; // Lua frame info. Takes 48-50 bits.
            struct {
                u8 ffid; // ID of FF function. 1 = Regular C function.
                // See isffunc@lj_obj.h
            } c_frame_data; // C or FF frame info. Takes 48-55 bits.
            struct {
                enum lua_unwind_error error_kind : 2; // Encountered error type
                u8 frame_gct : 4; // The GC type of this frame
            } invalid_frame_data; // Invalid frame info. Takes 48-53 bits.
        };
    } lua_data; // Tagged pointer with lua frame info
    u64 raw;    // Raw 64-bit value
};

//
//
//
//
//
//
//
// Additional newlines above to prevent redefinition of BTF_EXPORT
BTF_EXPORT(enum lua_context_type);
BTF_EXPORT(enum lua_unwind_error);

_Static_assert(LJ_ARCH_ENDIAN == LUAJIT_LE); // TODO: We currently support little-endian architectures only
_Static_assert(sizeof(union lua_symbol_key_data) ==
                   sizeof(((struct symbol_key *)0)->object_addr),
               "Union must match the size of struct symbol_key::object_addr");

// Lua functions do have line information in the range [0; 0x7fffff00).
// Using -1 as indication of C, FF or invalid frame.
static const i32 lua_line_start_not_used = -1;

struct lua_config {
    u32 version; // Version of LuaJIT. Encoded as (minor << 8) + (major << 16).
                 // See encodeVersion@lua.go
    u16 lj_bc_mode[GG_LEN_DDISP]; // LuaJIT bytecode mode array. Filled in
                                  // ParseLuaUnwinderConfig@lua.go
    u64 offset_g_to_l;            // `offsetof(global_State, cur_L)`
    u64 offset_g_to_dispatch;     // `GG_G2DISP`
};

/**
 * @brief Simple structure tracking the state of symbol buffer.
 * Simplifies writing to `struct symbol`.
 *
 */
struct symbol_state {
    struct symbol *symbol; // Pointer to the current symbol.
    char *caret;           // Current caret to the end of the buffer.
    size_t size;           // Written size.
};

struct lua_state {
    u32 pid;                  // PID of the current process.
    struct lua_config config; // Config of LuaJIT binary found in this process.

    struct interpreter_stack stack; // Call stack of current `lua_State`.
    struct symbol symbol;           // Temporary buffer for frame information.

    u64 dispatch_register; // Value of `r14`. This register holds pointer to
                           // GG_State->dispatch.
    u64 base_register; // Value of `rdx`. This register might have a hint about
                       // the actual L->base value.
    // u64 pc_register;    // Value of `rbx`. This register contains current
    // instruction executed by VM. Verify that interpreter is
    // running before use

    u64 L; // Current `lua_State*`.
    u64 G; // global_State*

    // buffer to read variable names
    char buffer[1024];
    bool jit;
};

BPF_MAP(lua_storage, BPF_MAP_TYPE_HASH, binary_id, struct lua_config,
        MAX_BINARIES)

struct lua_global_state_key {
    u32 pid;
};

struct lua_global_state_cache {
    u64 G; // global_State*
};

BPF_MAP(lua_global_state_storage, BPF_MAP_TYPE_HASH,
        struct lua_global_state_key, struct lua_global_state_cache,
        MAX_BINARIES)

struct lua_variable {
    // cannot use const char * or char * here:
    // Error: failed to process file "unwinder.release.elf": unsupported type:
    // *btf.Const unwinder.go:588:17: cannot use unsafe.Pointer(uintptr(ptr))
    // (value of type unsafe.Pointer) as *byte value in assignment
    u64 name;
    uint32_t startpc;
    uint32_t endpc;
};

BPF_MAP(lua_variables_storage, BPF_MAP_TYPE_PERCPU_ARRAY, u32,
        struct lua_variable, LJ_MAX_LOCVAR)
BPF_MAP(lua_upvalues_storage, BPF_MAP_TYPE_PERCPU_ARRAY, u32,
        struct lua_variable, LJ_MAX_UPVAL)
