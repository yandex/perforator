#pragma once

#include <bpf/bpf.h>

#include "types.h"

// namespace luajit

// Definitions are written in the same order as in original LuaJIT files.
// Hardcoded offsets are based on these flags:
// LJ_GC64 == 1
// LJ_FR2 == 1

// Config

enum {
    LUAJIT_LJ_FR2 = 1 // LJ_FR2
};

// Helpers

#define LUAJIT_GET_OFFSET(ptr, offset) ((void*)(((char*)(ptr)) + (offset)))

#define LUAJIT_READ_FROM_OFFSET(return_type, ptr, offset) BPF_PROBE_READ_USER_FROM((return_type*)LUAJIT_GET_OFFSET((ptr), (offset)))

#define LUAJIT_DEFINE_FIELD_GETTER(return_type, class_name, field_name, offset) \
    [[nodiscard]] static ALWAYS_INLINE return_type class_name##_get_##field_name(class_name* value) { \
        return LUAJIT_READ_FROM_OFFSET(return_type, value, (offset)); \
    }

// lj_obj.h

// Tagged value.
// TValue
typedef u64 luajit_tvalue;

static const u32 LUAJIT_LJ_TFUNC = (~8u);
static const u64 LUAJIT_LJ_GCVMASK = (1ULL << 47) - 1;

// GCstr
enum {
    LUAJIT_GC_STR_SIZEOF = 24, // sizeof(GCstr)
};

// GCproto
typedef struct luajit_gc_proto luajit_gc_proto;
enum {
    LUAJIT_GC_PROTO_SIZEOF = 104,   // sizeof(GCproto)
    LUAJIT_GC_PROTO_CHUNKNAME = 64, // GCRef chunkname;
    LUAJIT_GC_PROTO_FIRSTLINE = 72, // BCLine firstline;
    LUAJIT_GC_PROTO_NUMLINE = 76,   // BCLine numline;
};
LUAJIT_DEFINE_FIELD_GETTER(void*, luajit_gc_proto, chunkname, LUAJIT_GC_PROTO_CHUNKNAME);
LUAJIT_DEFINE_FIELD_GETTER(i32, luajit_gc_proto, firstline, LUAJIT_GC_PROTO_FIRSTLINE);
LUAJIT_DEFINE_FIELD_GETTER(i32, luajit_gc_proto, numline, LUAJIT_GC_PROTO_NUMLINE);

[[nodiscard]] static ALWAYS_INLINE const char* luajit_proto_chunknamestr(luajit_gc_proto* proto) {
    // #define strref(r)	(&gcref((r))->str)
    // #define proto_chunkname(pt)	(strref((pt)->chunkname))
    // #define strdata(s)	((const char *)((s)+1))
    // #define proto_chunknamestr(pt)	(strdata(proto_chunkname((pt))))
    return (char*)luajit_gc_proto_get_chunkname(proto) + LUAJIT_GC_STR_SIZEOF;
}

// GCfunc
typedef union luajit_gc_func luajit_gc_func;
enum {
    LUAJIT_GC_FUNC_FFID = 10, // uint8_t ffid;
    LUAJIT_GC_FUNC_PC = 32,   // MRef pc;
    LUAJIT_GC_FUNC_F = 40,    // lua_CFunction f;
};
LUAJIT_DEFINE_FIELD_GETTER(u8, luajit_gc_func, ffid, LUAJIT_GC_FUNC_FFID);
LUAJIT_DEFINE_FIELD_GETTER(void*, luajit_gc_func, pc, LUAJIT_GC_FUNC_PC);
LUAJIT_DEFINE_FIELD_GETTER(void*, luajit_gc_func, f, LUAJIT_GC_FUNC_F);

static const int LUAJIT_FF_LUA = 0;

[[nodiscard]] static ALWAYS_INLINE bool luajit_isluafunc(luajit_gc_func* function) {
    // #define isluafunc(fn)	((fn)->c.ffid == FF_LUA)
    return luajit_gc_func_get_ffid(function) == LUAJIT_FF_LUA;
}

[[nodiscard]] static ALWAYS_INLINE luajit_gc_proto* luajit_funcproto(luajit_gc_func* function) {
    // #define funcproto(fn) (GCproto *)(mref((fn)->l.pc, char)-sizeof(GCproto))
    return (luajit_gc_proto*)((char*)(luajit_gc_func_get_pc(function)) - LUAJIT_GC_PROTO_SIZEOF);
}

enum {
    LUAJIT_VM_STATE_INTERPRETING = 0, // LJ_VMST_INTERP
};

// Global state, shared by all threads of a Lua universe.
// global_State
typedef struct luajit_global_state luajit_global_state;

/**
 * @brief Get `luajit_global_state->vmstate`.
 *
 * @param global_state Global state.
 * @param config Current process config.
 * @return int Current VM state.
 */
[[nodiscard]] static ALWAYS_INLINE int luajit_global_state_get_vm_state(luajit_global_state* global_state, const struct lua_config* config) {
    return LUAJIT_READ_FROM_OFFSET(int, global_state, config->offset_g_to_vm_state);
}

/**
 * @brief Get `luajit_global_state->cur_L`.
 *
 * @param global_state Global state.
 * @param config Current process config.
 * @return lua_State* Currently executing `lua_State`.
 */
[[nodiscard]] static ALWAYS_INLINE struct luajit_state* luajit_global_state_get_current_lua_state(luajit_global_state* global_state, const struct lua_config* config) {
    return LUAJIT_READ_FROM_OFFSET(struct luajit_state*, global_state, config->offset_g_to_l);
}

/**
 * @brief Get `global_state->jit_base`.
 *
 * @param global_state Global state.
 * @param config Current process config.
 * @return const luajit_tvalue* Currently executing `lua_State`.
 */
[[nodiscard]] static ALWAYS_INLINE const luajit_tvalue* luajit_global_state_get_jit_base(luajit_global_state* global_state, const struct lua_config* config) {
    // `jit_base` is always next to `cur_L` in `global_State`.
    // They both are 64 bit values, hence they have no additional alignment.
    // It's easier to rely on that rather than decoding it from the assembly.
    return LUAJIT_READ_FROM_OFFSET(const luajit_tvalue*, global_state, config->offset_g_to_l + sizeof(struct luajit_state*));
}

/**
 * @brief Get address of `global_State` from dispatch address. See `GG_G2DISP`.
 *
 * Doesn't dereference dispatch address.
 *
 * @param dispatch Dispatch address.
 * @param config Current process config.
 * @return global_State* Pointer to global state.
 */
[[nodiscard]] static ALWAYS_INLINE luajit_global_state* luajit_get_global_state_from_dispatch(u64 dispatch, const struct lua_config* config) {
    return LUAJIT_GET_OFFSET(dispatch, -config->offset_g_to_dispatch);
}

// Per-thread state object.
// lua_State
typedef struct luajit_state luajit_state;
enum {
    LUAJIT_LUA_STATE_GLREF = 16,    // MRef glref;
    LUAJIT_LUA_STATE_BASE = 32,     // TValue *base;
    LUAJIT_LUA_STATE_MAXSTACK = 48, // MRef maxstack;
    LUAJIT_LUA_STATE_STACK = 56,    // MRef stack;
    LUAJIT_LUA_STATE_CFRAME = 80,   // void *cframe;
};
LUAJIT_DEFINE_FIELD_GETTER(luajit_global_state*, luajit_state, glref, LUAJIT_LUA_STATE_GLREF); // Combines `G(L)` macro
LUAJIT_DEFINE_FIELD_GETTER(luajit_tvalue*, luajit_state, base, LUAJIT_LUA_STATE_BASE);
LUAJIT_DEFINE_FIELD_GETTER(luajit_tvalue*, luajit_state, maxstack, LUAJIT_LUA_STATE_MAXSTACK);
LUAJIT_DEFINE_FIELD_GETTER(luajit_tvalue*, luajit_state, stack, LUAJIT_LUA_STATE_STACK);
LUAJIT_DEFINE_FIELD_GETTER(void*, luajit_state, cframe, LUAJIT_LUA_STATE_CFRAME);

// GCobj
typedef union luajit_gc_obj luajit_gc_obj;

[[nodiscard]] static ALWAYS_INLINE u32 luajit_itype(const luajit_tvalue* value) {
    // #define itype(o)	((uint32_t)((o)->it64 >> 47))
    return (u32)(BPF_PROBE_READ_USER_FROM((i64*)value) >> 47);
}

[[nodiscard]] static ALWAYS_INLINE bool luajit_tvisfunc(const luajit_tvalue* value) {
    // #define tvisfunc(o)	(itype(o) == LJ_TFUNC)
    return luajit_itype(value) == LUAJIT_LJ_TFUNC;
}

[[nodiscard]] static ALWAYS_INLINE luajit_gc_obj* luajit_gcval(const luajit_tvalue* value) {
    // #define gcrefu(r)	((r).gcptr64)
    // #define gcval(o)	((GCobj *)(gcrefu((o)->gcr) & LJ_GCVMASK))
    return (luajit_gc_obj*)(BPF_PROBE_READ_USER_FROM((u64*)value) & LUAJIT_LJ_GCVMASK);
}

// lj_bc.h

[[nodiscard]] static ALWAYS_INLINE u32 luajit_bc_a(u32 instruction) {
    // #define bc_a(i)		((BCReg)(((i)>>8)&0xff))
    return (u32)((instruction >> 8) & 0xff);
}

// lj_frame.h

static const int LUAJIT_FRAME_LUA = 0;
static const int LUAJIT_FRAME_VARG = 3;
static const int LUAJIT_FRAME_TYPE = 3;
static const int LUAJIT_FRAME_P = 4;
static const int LUAJIT_FRAME_TYPEP = (LUAJIT_FRAME_TYPE | LUAJIT_FRAME_P);

[[nodiscard]] static ALWAYS_INLINE luajit_gc_obj* luajit_frame_gc(const luajit_tvalue* frame) {
    // #define frame_gc(f)		(gcval((f)-1))
    return luajit_gcval(frame - LUAJIT_LJ_FR2);
}

[[nodiscard]] static ALWAYS_INLINE i64 luajit_frame_ftsz(const luajit_tvalue* frame) {
    // #define frame_ftsz(f)		((ptrdiff_t)(f)->ftsz)
    return BPF_PROBE_READ_USER_FROM((i64*)frame);
}

[[nodiscard]] static ALWAYS_INLINE u32* luajit_frame_pc(const luajit_tvalue* frame) {
    // #define frame_pc(f)		((const BCIns *)frame_ftsz(f))
    return (u32*)luajit_frame_ftsz(frame);
}

[[nodiscard]] static ALWAYS_INLINE i64 luajit_frame_type(const luajit_tvalue* frame) {
    // #define frame_type(f)		(frame_ftsz(f) & FRAME_TYPE)
    return luajit_frame_ftsz(frame) & LUAJIT_FRAME_TYPE;
}

[[nodiscard]] static ALWAYS_INLINE i64 luajit_frame_typep(const luajit_tvalue* frame) {
    // #define frame_typep(f)		(frame_ftsz(f) & FRAME_TYPEP)
    return luajit_frame_ftsz(frame) & LUAJIT_FRAME_TYPEP;
}

[[nodiscard]] static ALWAYS_INLINE bool luajit_frame_islua(const luajit_tvalue* frame) {
    // #define frame_islua(f)		(frame_type(f) == FRAME_LUA)
    return luajit_frame_type(frame) == LUAJIT_FRAME_LUA;
}

[[nodiscard]] static ALWAYS_INLINE bool luajit_frame_isvarg(const luajit_tvalue* frame) {
    // #define frame_isvarg(f)		(frame_typep(f) == FRAME_VARG)
    return luajit_frame_typep(frame) == LUAJIT_FRAME_VARG;
}

[[nodiscard]] static ALWAYS_INLINE i64 luajit_frame_sized(const luajit_tvalue* frame) {
    // #define frame_sized(f)		(frame_ftsz(f) & ~FRAME_TYPEP)
    return luajit_frame_ftsz(frame) & ~LUAJIT_FRAME_TYPEP;
}

[[nodiscard]] static ALWAYS_INLINE const luajit_tvalue* luajit_frame_prevl(const luajit_tvalue* frame) {
    // #define frame_prevl(f)		((f) - (1+LJ_FR2+bc_a(frame_pc(f)[-1])))
    return (frame) - (1 + LUAJIT_LJ_FR2 + luajit_bc_a(BPF_PROBE_READ_USER_FROM(&luajit_frame_pc(frame)[-1])));
}

[[nodiscard]] static ALWAYS_INLINE const luajit_tvalue* luajit_frame_prevd(const luajit_tvalue* frame) {
    // #define frame_prevd(f)		((TValue *)((char *)(f) - frame_sized(f)))
    return (const luajit_tvalue*)((char*)(frame)-luajit_frame_sized(frame));
}
