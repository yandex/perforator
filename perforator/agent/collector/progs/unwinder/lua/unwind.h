#pragma once

#include <bpf/bpf.h>
#include <linux/errno.h>

#include "../binary.h"
#include "../interpreter/types.h"
#include "../metrics.h"
#include "../process.h"

#define LUAJIT_DISABLE_FFI

/*
 * Non-CO-RE variant of BPF_CORE_READ_USER that requires just an address to read
 * instead of struct field.
 *
 * To get correct type, cast the expression to the required type pointer
 *
 * As no CO-RE relocations are emitted, source types can be arbitrary and are
 * not restricted to kernel types only.
 */
#define BPF_PROBE_READ_USER_POINTER(src)                                       \
    ({                                                                         \
        typeof(*(src)) __r;                                                    \
        bpf_probe_read_user(&__r, sizeof(*(src)), (src));                      \
        __r;                                                                   \
    })

#include "trace.h"
#include "types.h"

#define NODISCARD __attribute__((warn_unused_result))
#define MIN(a, b) (a) > (b) ? (b) : (a)

static ALWAYS_INLINE void read_value(void *value, u32 size, const void *ptr,
                                     char c) {
    LUA_TRACE("Trying to read %px (%c)", ptr, c);

    int err = bpf_probe_read_user(value, size, ptr);

    if (err != 0) {
        metric_increment(METRIC_LUA_DEREF_ERROR_COUNT);
        LUA_TRACE("failed to read %px: %d", ptr, err);
    }
}

// clang-format off
#define READ_SAFE(variable_name, address, c)                                                 \
  _Pragma("clang diagnostic push");                                                          \
  _Pragma("clang diagnostic ignored \"-Wincompatible-pointer-types-discards-qualifiers\"");  \
  typeof(*(address)) variable_name;                                                          \
  read_value(&variable_name, sizeof(typeof(*(address))), (address), (c));                    \
  _Pragma("clang diagnostic pop")
// clang-format on

// Config

/**
 * @brief Get config for the found LuaJIT binary
 *
 * @param state Lua unwind state.
 * @param process_info Information about current process, and specifically the
 * binary where LuaJIT is located.
 * @returns `true` if config was successfully found.
 */
static ALWAYS_INLINE bool
lua_state_config_get(struct lua_state *state,
                     struct process_info *process_info) {
    binary_id id = process_info->lua_binary.id;
    struct lua_config *config = bpf_map_lookup_elem(&lua_storage, &id);

    if (config == NULL) {
        return false;
    }

    // Just a sanity check we read lj_bc_mode correctly for go land
    // if (config->lj_bc_mode[0] != 0x3183) {
    // metric_increment(METRIC_LUA_WRONG_BC_MODE_COUNT);
    // return false;
    // }

    state->config = *config;
    return true;
}

// Frame

/**
 * @brief Gets current uncommitted frame.
 *
 * @param state Lua unwind state.
 * @private
 * @return struct interpreter_frame* Current uncommited frame.
 */
static ALWAYS_INLINE struct interpreter_frame *
lua_frame_get_current(struct lua_state *state) {
    return &state->stack.frames[state->stack.len];
}
/**
 * @brief
 *
 * @param state
 * @param data
 * @param linestart
 */
static ALWAYS_INLINE void lua_frame_write(struct lua_state *state,
                                          union lua_symbol_key_data data,
                                          i32 linestart) {
    __auto_type key = &lua_frame_get_current(state)->symbol_key;

    key->object_addr = data.raw;
    key->pid = state->pid;
    key->linestart = linestart;
}

static bool string_compare(const char *s1, const char *s2, size_t len) {
    for (size_t i = 0; i != 64 && i != len; ++i) {
        if (s1[i] != s2[i]) {
            //LUA_TRACE("%c != %c", s1[i], s2[i]);
            return false;
        }
    }

    return true;
}

/**
 * @brief Finishes frame by pushing symbol info to the map and incrementing
 * stack size.
 *
 * @param state Lua unwind state.
 */
static ALWAYS_INLINE void lua_frame_commit(struct lua_state *state) {
    LUA_TRACE("frame commit object_addr=%px linestart=%d",
              lua_frame_get_current(state)->symbol_key.object_addr,
              lua_frame_get_current(state)->symbol_key.linestart);
    LUA_TRACE("symbol %d %d %s", state->symbol.name_length,
              state->symbol.filename_length, state->symbol.data);

    struct symbol *cached_symbol = (struct symbol *)bpf_map_lookup_elem(
        &interpreter_symbols, lua_frame_get_current(state));

    if (!cached_symbol) {
        LUA_TRACE("cached_symbol = nullptr");
    } else if (state->symbol.name_length == cached_symbol->name_length &&
               state->symbol.filename_length ==
                   cached_symbol->filename_length &&
               string_compare(state->symbol.data, cached_symbol->data,
                              state->symbol.name_length +
                                  state->symbol.filename_length)) {
        LUA_TRACE("symbol == cached symbol");
    } else {
        metric_increment(METRIC_LUA_CACHE_MISMATCH_COUNT);
        LUA_TRACE("symbol MISMATCH cached symbol");
        LUA_TRACE("!!! name_length    : %d | %d", state->symbol.name_length,
                  cached_symbol->name_length);
        LUA_TRACE("!!! filename_length: %d | %d", state->symbol.filename_length,
                  cached_symbol->filename_length);
        LUA_TRACE("!!! data           : %s | %s", state->symbol.data,
                  cached_symbol->data);
    }

    int result =
        bpf_map_update_elem(&interpreter_symbols, lua_frame_get_current(state),
                            &state->symbol, BPF_ANY);

    if (result != 0) {
        // TODO
    }

    ++state->stack.len;
}

// Stack

/**
 * @brief Resets the stack to empty state.
 *
 * @param state Lua unwind state.
 */
static ALWAYS_INLINE void lua_stack_reset(struct lua_state *state) {
    state->stack.len = 0;
}

// Symbol

/**
 * @brief Clamps size to be in bound of symbol buffer size. This helps verifier
 * to understand possible range of values.
 *
 * @param size Size to clamp.
 * @return size_t Clamped size.
 */
static ALWAYS_INLINE size_t lua_symbol_clamp_size(size_t size) { // maybe unused
    return size & INTERPRETER_SYMBOL_STRING_LENGTH_VERIFIER_MASK;
}

/**
 * @brief Prepares the symbol. This should be called once.
 * @param state Lua unwind state.
 */
static ALWAYS_INLINE void lua_symbol_prepare(struct lua_state *state) {
    state->symbol.codepoint_size = 1; // Lua strings are always utf-8
}

/**
 * @brief Resets the symbol to empty state. This is called before writing a new
 * symbol.
 *
 * @param state Lua unwind state.
 * @return struct symbol_state New symbol state. Use this to write data into it.
 */
static ALWAYS_INLINE struct symbol_state
lua_symbol_new(struct lua_state *state) {
    state->symbol.name_length = 0;
    state->symbol.filename_length = 0;

    return (struct symbol_state){
        .symbol = &state->symbol, .caret = state->symbol.data, .size = 0};
}

static ALWAYS_INLINE size_t
lua_symbol_get_remaining_capacity(struct symbol_state *state) {
    return SYMBOL_BUFFER_SIZE - state->size;
}

/**
 * @brief Sets the length of symbol name.
 * Use this after finishing writing data.
 *
 * @param state Symbol state.
 */
static ALWAYS_INLINE void lua_symbol_name_commit(struct symbol_state *state) {
    state->symbol->name_length = state->size;
}

/**
 * @brief Sets the length of symbol filename.
 * Use this after finishing writing data.
 * @note This must be used after writing symbol name. @see struct symbol.
 *
 * @param state Symbol state.
 */
static ALWAYS_INLINE void
lua_symbol_filename_commit(struct symbol_state *state) {
    state->symbol->filename_length = state->size - state->symbol->name_length;
}

/**
 * @brief Writes one character to symbol buffer.
 *
 * @param state Symbol state.
 * @param symbol Character to write.
 */
static ALWAYS_INLINE void lua_symbol_append_char(struct symbol_state *state,
                                                 char symbol) {
    state->caret++[0] = symbol;
    ++state->size;
}

/**
 * @brief Writes string to symbol buffer.
 * @note The size of the string must be known at compile-time.
 * @see LUA_SYMBOL_APPEND_LITERAL
 *
 * @param state Symbol state.
 * @param string String to write.
 * @param string_size Size of the string.
 */
static ALWAYS_INLINE void lua_symbol_append_string(struct symbol_state *state,
                                                   const char *string,
                                                   size_t string_size) {
    memcpy(state->caret, string, string_size);

    state->caret += string_size;
    state->size += string_size;
}

/**
 * @brief Writes literal or char array to symbol buffer.
 * @see lua_symbol_append_string
 * @note Writes without a NULL terminator.
 *
 * @param state Symbol state.
 * @param literal Literal.
 */
#define LUA_SYMBOL_APPEND_LITERAL(state, literal)                              \
    lua_symbol_append_string((state), (literal), sizeof(literal) - 1)

/**
 * @brief Writes error message to symbol buffer
 * @note Writes without a NULL terminator.
 *
 * @param state Symbol state.
 */
static ALWAYS_INLINE void lua_symbol_append_fail(struct symbol_state *state) {
    const char kInvalidString[] = "<failed to read>";

    LUA_SYMBOL_APPEND_LITERAL(state, kInvalidString);
}

/**
 * @brief Reads user-space string and writes it to symbol buffer.
 * @note Writes without a NULL terminator.
 *
 * @param state Symbol state.
 * @param unsafe_input User-space string pointer.
 * @return bool `true` if reading was successful.
 */
static ALWAYS_INLINE bool
lua_symbol_append_user_string(struct symbol_state *state,
                              const char *unsafe_input) {
    u32 length = lua_symbol_get_remaining_capacity(state);
    // "R2 unbounded memory access, use 'var &= const' or 'if (var < const)'"}
    // length &= INTERPRETER_SYMBOL_STRING_LENGTH_VERIFIER_MASK;

    long status = bpf_probe_read_user_str(state->caret, length, unsafe_input);

    if (status <= 0) {
        lua_symbol_append_fail(state);
        return false;
    }

    // Without NULL terminator
    state->caret += status - 1;
    state->size += status - 1;
    return true;
}

/**
 * @brief Writes BCLine to symbol buffer in base 10.
 *
 * @param state Symbol state.
 * @param line Line to write.
 */
static void lua_symbol_append_bcline(struct symbol_state *state, BCLine line) {
    size_t remaining_capacity = lua_symbol_get_remaining_capacity(state);

    uint32_t unsigned_line;
    bool is_negative;

    if (line < 0) {
        is_negative = true;
        unsigned_line = -line;
    } else {
        is_negative = false;
        unsigned_line = line;
    }

    unsigned int digits = 0;

    // Counting digits
    {
        uint32_t copy = unsigned_line;

        do {
            ++digits;
            copy /= 10;
        } while (digits < 10 && copy != 0);
    }

    if (!(remaining_capacity - is_negative)) {
        LUA_TRACE("!remaining_capacity");
        return;
    }

    // Writing minus
    if (is_negative) {
        lua_symbol_append_char(state, '-');
        --remaining_capacity;
    }

    int digits_copy = digits;
    int count = 0;

    // Writing digits
    while (digits_copy) {
        unsigned int pos = --digits_copy;

        if (pos < remaining_capacity) {
            state->caret[pos] = '0' + (unsigned_line % 10);
            --remaining_capacity;
            ++count;
        }

        unsigned_line /= 10;
    }

    state->caret += count;
    state->size += count;
}

// /* Convert a nibble (0..15) to hex char */
// static ALWAYS_INLINE char nibble_to_hex(uint8_t nibble) {
//   return ;
// }

/**
 * @brief Writes address to symbol buffer in format `%#x`.
 *
 * @param state Symbol state.
 * @param address Address to write.
 */
static void lua_symbol_append_address(struct symbol_state *state,
                                      void *address) {
    static const size_t kMaxDigits = sizeof(uintptr_t);
    // size_t remaining_capacity = lua_symbol_get_remaining_capacity(state);
    // unsigned int digits = 0;

    LUA_SYMBOL_APPEND_LITERAL(state, "0x");

    uintptr_t address_as_number = (uintptr_t)address;

    for (size_t pos = 0, offset = (kMaxDigits - 1) * 4; pos < kMaxDigits;
         ++pos, offset -= 4) {
        uint8_t nibble = (address_as_number >> offset) & 0x0f;
        state->caret[pos] =
            (nibble < 10) ? ('0' + nibble) : ('a' + (nibble - 10));
    }

    state->caret += kMaxDigits;
    state->size += kMaxDigits;
}

/* Invalid bytecode position. */
#define NO_BCPOS (~(BCPos)0)

/* Get line number for a bytecode position. */
static ALWAYS_INLINE BCLine bpf_lj_debug_line(GCproto *pt, BCPos pc) {
    const void *lineinfo = proto_lineinfo(pt);
    __auto_type sizebc = BPF_PROBE_READ_USER(pt, sizebc);
    __auto_type firstline = BPF_PROBE_READ_USER(pt, firstline);
    __auto_type numline = BPF_PROBE_READ_USER(pt, numline);

    if (pc <= sizebc && lineinfo) {
        BCLine first = firstline;

        if (pc == sizebc) {
            return first + numline;
        }

        if (pc-- == 0) {
            return first;
        }

        if (numline < 256) {
            READ_SAFE(offset, ((const uint8_t *)lineinfo) + pc, ':');
            return first + offset;
        } else if (numline < 65536) {
            READ_SAFE(offset, ((const uint16_t *)lineinfo) + pc, ':');
            return first + offset;
        } else {
            READ_SAFE(offset, ((const uint32_t *)lineinfo) + pc, ':');
            return first + offset;
        }
    }

    return 0;
}

// original function - debug_framepc, lj_debug.c
static ALWAYS_INLINE BCPos bpf_debug_framepc(struct lua_state *state,
                                             GCfunc *fn, cTValue *nextframe) {
    lua_State *L = (lua_State *)state->L;

    if (!isluafunc(fn)) { /* Cannot derive a PC for non-Lua functions. */
        return NO_BCPOS;
    }

    const BCIns *ins;

    if (nextframe == NULL) { /* Lua function on top. */
        void *cf = cframe_raw(BPF_PROBE_READ(L, cframe));

        if (cf == NULL || (char *)cframe_pc(cf) == (char *)cframe_L(cf)) {
            return NO_BCPOS;
        }

        ins = cframe_pc(cf); /* Only happens during error/hook handling. */

        if (!ins) {
            return NO_BCPOS;
        }
    } else {
        if (frame_islua(nextframe)) {
            ins = frame_pc(nextframe);
        } else if (frame_iscont(nextframe)) {
            ins = frame_contpc(nextframe);
        } else {
            /* Lua function below errfunc/gc/hook: find cframe to get the PC. */
            // TODO
            return NO_BCPOS;
        }
    }

    GCproto *pt = funcproto(fn);
    BCPos pos = proto_bcpos(pt, ins) - 1;

#if LJ_HASJIT
    if (pos >
        BPF_PROBE_READ_USER(
            pt, sizebc)) { /* Undo the effects of lj_trace_exit for JLOOP. */
        if (bc_isret(bc_op(BPF_PROBE_READ_USER_POINTER(&ins[-1])))) {
            return NO_BCPOS; /* Punt in case of stack overflow for stitched
                                trace. */
        }

        GCtrace *T =
            (GCtrace *)((char *)(ins - 1) - offsetof(GCtrace, startins));

        return proto_bcpos(pt,
                           mref(BPF_PROBE_READ_USER(T, startpc), const BCIns));
    }
#endif

    return pos;
}

// original function - debug_frameline, lj_debug.c
static ALWAYS_INLINE BCLine bpf_debug_frameline(struct lua_state *state,
                                                GCfunc *fn,
                                                cTValue *nextframe) {
    BCPos pc = bpf_debug_framepc(state, fn, nextframe);

    if (pc != NO_BCPOS) {
        return bpf_lj_debug_line(funcproto(fn), pc);
    }

    return -1;
}

static void lua_push_invalid_frame(struct lua_state *state,
                                   enum lua_unwind_error error, uint8_t gct) {
    union lua_symbol_key_data data = {.lua_data = {.ptr = 0,
                                                   .invalid_frame_data = {
                                                       .error_kind = error,
                                                       .frame_gct = gct,
                                                   }}};

    lua_frame_write(state, data, lua_line_start_not_used);
    lua_symbol_new(state);
    lua_frame_commit(state);
}

/* Get name of a local variable from slot number and PC. */
// original function - debug_varname, lj_debug.c

static NOINLINE uint64_t read_varname(struct lua_state *state, uint64_t p_value,  int i, BCPos lastpc){
    uint64_t p = p_value;
    uint64_t success_flag = 0;

    if(p == 0){
        goto return_result;
    }
    const char *name = (const char*)p;
    uint8_t vn;

    int status = bpf_probe_read_user(&vn, sizeof(vn), (const void*)p);
    if(status != 0){
        goto return_result;
    }

    BCPos startpc, endpc;
    if (vn < VARNAME__MAX) {
        // pre-defined variable, name index is stored as an integer constant
        // (uint8_t)
        if (vn == VARNAME_END) {
            /* End of varinfo. */
            goto return_result;
        }
        p += 1;
    } else {
        // an actual variable name stored as string
        status = bpf_probe_read_user_str(state->buffer, sizeof(state->buffer) , (const void*)p);
        if (status < 0) {
            goto return_result;
        }
        p += status;
    }
    uint64_t result = lj_buf_ruleb128_new(p);

    if (!(result & 0x800000000000000)) {
        goto return_result;
    }
    p += (uint8_t)(result>>32) +1;
    lastpc = startpc = lastpc + (uint32_t)result;

    result = lj_buf_ruleb128_new(p);
    if (!(result & 0x800000000000000)) {
        goto return_result;
    }
    p += (uint8_t)(result>>32) +1;
    endpc = startpc + (uint32_t)result;

    struct lua_variable *variable = bpf_map_lookup_elem(&lua_variables_storage, &i);
    if (variable) {
        variable->name = (u64)name;
        variable->startpc = startpc;
        variable->endpc = endpc;
    }
    success_flag = 0x8000000000000000;

  return_result:
    return success_flag + ((uint64_t)(p - p_value) <<32) +lastpc;
}

static ALWAYS_INLINE int bpf_debug_varname(struct lua_state *state, GCproto *pt,
                                           BCPos pc) {
    const char *p = (const char *)proto_varinfo(pt);
    if (!p) {
        return 0;
    }
    if(state == NULL){
        return 0;
    }
    BCPos lastpc = 0;
    // original function uses an infinite loop to scan variables block
    // here in BPF program use some reasonable default - 200 local variables
    // (LJ_MAX_LOCVAR)

    int i = 0;
    for (; i < 40 /*LJ_MAX_LOCVAR*/; ++i) {

        uint64_t result = read_varname( state, (uint64_t)p, i, lastpc);
        p += (uint8_t)(result >> 32);
        if(!(result & 0x8000000000000000)){
            return i;
        }
        lastpc = (uint32_t)result;
    }
    return i + 1;
}

static __always_inline int bpf_debug_uvname(struct lua_state *L, GCproto *pt)
{
    const uint8_t *p = proto_uvinfo(pt);
    if (!p)
        return 0;

    int i = 0;
    int status;

    #define READ_UV(n) \
        { \
            const char *name_##n = (const char *)p; \
            status = bpf_probe_read_user_str(L->buffer, sizeof(L->buffer), p); \
            if (status <= 0) goto done; \
            p += status; \
            struct lua_variable *uv = bpf_map_lookup_elem(&lua_upvalues_storage, &i); \
            if (uv) uv->name = (uint64_t)name_##n; \
            i++; \
        }

    READ_UV(0);  READ_UV(1);  READ_UV(2);  READ_UV(3);  READ_UV(4);
    READ_UV(5);

#undef READ_UV

done:
    return i;
}

// original function - lj_debug_slotname, lj_debug.c
static ALWAYS_INLINE const char *
bpf_debug_slotname(struct lua_state *state, struct symbol_state *symbol,
                   GCproto *pt, const BCIns *ip, BCReg slot,
                   enum lua_context_type *context_type) {
    const char *lname;

    BCPos pc = proto_bcpos(pt, ip);
    BCIns *bc = proto_bc(pt);
    uint16_t *lj_bc_mode = state->config.lj_bc_mode;
    // local variable block is an invariant of loop and can be read into BPF
    // array once while walking the loop we only identify the slot index within
    // variables array
    int count_variables = bpf_debug_varname(state, pt, pc);
    int count_upvalues = bpf_debug_uvname(state, pt);
    //int count_upvalues = 0;
    // local variable is only read on "start" and on "restart" (BCmov "ra ==
    // slot" condition)
    bool read_local = true;

    for (size_t i = 0; i < 10; ++i) {
        if (read_local) {
            struct lua_variable *variable =
                bpf_map_lookup_elem(&lua_variables_storage, &slot);
            if (variable && slot < count_variables) {
                if (variable->startpc <= pc && pc < variable->endpc) {
                    *context_type = LUA_CONTEXT_LOCAL;
                    return (const char *)variable->name;
                }
            }
            read_local = false;
        }

        --ip;

        if (ip < bc) {
            return NULL;
        }

        BCIns ins, previous_ins;
        int status = bpf_probe_read_user(&ins, sizeof(ins), ip);

        BCOp op = bc_op(ins);
        BCReg ra = bc_a(ins);
        BCMode mode_a = bcmode_a(op);
        BCReg d = bc_d(ins);
        if (mode_a == BCMbase) {
            if (slot >= ra && (op != BC_KNIL || slot <= d)) {
                return NULL;
            }

        } else if (mode_a == BCMdst && ra == slot) {
            switch (op) {
            case BC_MOV:
                if (ra == slot) {
                    slot = d;
                    read_local = true;
                }
                break;
            case BC_GGET: {
                ptrdiff_t v = ~(ptrdiff_t)bc_d(ins);
                GCobj *kgc = proto_kgc(pt, v);
                GCstr *gco = gco2str(kgc);
                const char *s = strdata(gco);
                *context_type = LUA_CONTEXT_GLOBAL;
                return s;
            }
            case BC_TGETS: {
                ptrdiff_t v = ~(ptrdiff_t)bc_c(ins);
                GCobj *kgc = proto_kgc(pt, v);
                GCstr *gco = gco2str(kgc);
                const char *s = strdata(gco);

                int status =
                    bpf_probe_read_user(&previous_ins, sizeof(ins), ip - 1);
                if (bc_op(previous_ins) == BC_MOV &&
                    bc_a(previous_ins) == ra + 1 + LJ_FR2 &&
                    bc_d(previous_ins) == bc_b(ins)) {
                    *context_type = LUA_CONTEXT_METHOD;
                } else {
                    *context_type = LUA_CONTEXT_FIELD;
                }
                return s;
            }
            case BC_UGET: {
                struct lua_variable *upvalue =
                    bpf_map_lookup_elem(&lua_upvalues_storage, &d);
                if (upvalue && d < count_upvalues) {
                    *context_type = LUA_CONTEXT_UPVALUE;
                    return (const char *)upvalue->name;
                } else {
                    return NULL;
                }
            }
            default:
                return NULL;
            }
        }
    }
    return NULL;
}

// original function - lj_debug_funcname, lj_debug.c
static ALWAYS_INLINE const char *
bpf_debug_funcname(struct symbol_state *symbol, struct lua_state *state,
                   cTValue *frame, enum lua_context_type *context_type) {
    *context_type = LUA_CONTEXT_UNKNOWN;

    lua_State *L = (lua_State *)state->L;
    global_State *g = G(L);
    cTValue *bot = tvref(BPF_PROBE_READ_USER(L, stack)) + LJ_FR2;

    if (frame <= bot) {
        return NULL;
    }

    if (frame_isvarg(frame)) {
        frame = frame_prevd(frame);
    }

    cTValue *pframe = frame_prev(frame);
    GCfunc *fn = frame_func(pframe);

    BCPos pc = bpf_debug_framepc(state, fn, frame);

    if (pc == NO_BCPOS) {
        return NULL;
    }

    GCproto *pt = funcproto(fn);
    __auto_type sizebc = BPF_PROBE_READ_USER(pt, sizebc);
    // const BCIns *ip = &proto_bc(pt)[check_exp(pc < sizebc, pc)];
    BCIns *bc = proto_bc(pt);
    BCIns *ip = bc + pc;
    BCIns ins;

    int status = bpf_probe_read_user(&ins, sizeof(ins), ip);
    if (status != 0) {
        return NULL;
    }

    BCOp op = bc_op(ins);

    uint16_t *lj_bc_mode = state->config.lj_bc_mode;
    MMS mm = bcmode_mm(op);

    if (mm == MM_call) {
        BCReg slot = bc_a(ins);

        if (op == BC_ITERC) {
            slot -= 3;
        }

        return bpf_debug_slotname(state, symbol, pt, ip, slot, context_type);
    } else if (mm != MM__MAX) {
        *context_type = LUA_CONTEXT_METAMETHOD;

        return strdata(mmname_str(g, mm));
    }

    return NULL;
}

NODISCARD static bool lua_push_lua_frame(struct lua_state *state,
                                         GCfunc *function, cTValue *frame,
                                         cTValue *nextframe) {
    GCproto *proto = funcproto(function);

    if (!proto) {
        LUA_TRACE("lua_push_lua_frame -> !proto");
        return false;
    }

    BCLine line_defined = bpf_debug_frameline(state, function, nextframe);

    if (line_defined == -1) {
        line_defined = BPF_PROBE_READ_USER(proto, firstline);
    }

    struct symbol_state symbol = lua_symbol_new(state);
    enum lua_context_type context_type;
    const char *funcname =
        bpf_debug_funcname(&symbol, state, frame, &context_type);

    union lua_symbol_key_data data = {
        .lua_data = {.ptr = (u64)proto,
                     .lua_frame_data = {.context_type = context_type}}};

    lua_frame_write(state, data, line_defined);

    // ar->what = (firstline || !pt->numline) ? "Lua" : "main";
    // if (*ar.what == 'm') { lua_pushliteral(L, " in main chunk");
    if ((BPF_PROBE_READ_USER(proto, firstline) ||
         !BPF_PROBE_READ_USER(proto, numline))) {
        if (funcname) {
            lua_symbol_append_user_string(&symbol, funcname);
        } else {
            LUA_SYMBOL_APPEND_LITERAL(&symbol, "<no name>");
        }
    } else {
        LUA_SYMBOL_APPEND_LITERAL(&symbol, "in main chunk");
    }

    lua_symbol_name_commit(&symbol);

    const char *filename = strdata(proto_chunkname(proto));

    lua_symbol_append_user_string(&symbol, filename);
    lua_symbol_filename_commit(&symbol);

    lua_frame_commit(state);

    return true;
}

NODISCARD static bool lua_push_c_frame(struct lua_state *state,
                                       GCfunc *function) {
    union lua_symbol_key_data data = {
        .lua_data = {
            .ptr = (u64)BPF_PROBE_READ_USER(function, c.f),
            .c_frame_data = {.ffid = BPF_PROBE_READ_USER(function, c.ffid)}}};

    lua_frame_write(state, data, lua_line_start_not_used);
    lua_symbol_new(state);
    lua_frame_commit(state);

    return true;
}

static bool lua_get_function_info(struct lua_state *state, cTValue *frame,
                                  cTValue *nextframe) {
    if (!frame) {
        lua_push_invalid_frame(state, LUA_UNWIND_ERROR_FRAME_IS_NULL, 0xF);
        metric_increment(METRIC_LUA_FRAME_IS_NULL_COUNT);

        return false;
    }

    GCfunc *fn = frame_func(frame);

    if (!fn) {
        lua_push_invalid_frame(state, LUA_UNWIND_ERROR_GCFUNC_IS_NULL, 0xF);
        metric_increment(METRIC_LUA_FUNCTION_IS_NULL_COUNT);

        return false;
    }

    // Notes
    // #define frame_delta(f)		(frame_ftsz(f) >> 3)
    // static void *err_unwind(lua_State *L, void *stopcf, int errcode)
    // lua_State Fake FF_C for curr_funcisL() on dummy frames.
    // else if (co->base > tvref(co->stack)+1+LJ_FR2) s = "normal";
    // coroutines
    // else if (co->top == co->base) s = "dead";
    // check if C func: if (op == BC_FUNCC || op == BC_FUNCCW)

    if (!tvisfunc(frame - LJ_FR2) ||
        BPF_PROBE_READ_USER(fn, c.gct) != ~LJ_TFUNC) {
        LUA_TRACE("!!! bad frame function got %u %u", ~itype(frame - LJ_FR2),
                  BPF_PROBE_READ_USER(fn, c.gct));
        lua_push_invalid_frame(state, LUA_UNWIND_ERROR_FRAME_IS_NOT_FUNC,
                               BPF_PROBE_READ_USER(fn, c.gct));
        metric_increment(METRIC_LUA_BROKEN_FRAME_COUNT);
        return false;
    }

    if (isluafunc(fn)) {
        return lua_push_lua_frame(state, fn, frame, nextframe);
    } else if (iscfunc(fn)) {
        return lua_push_c_frame(state, fn);
    }

    return lua_push_c_frame(state, fn);
}

// code is taken from lj_debug_frame, lj_debug.c
// almost the same code is also re-used in lj_debug_funcname
static ALWAYS_INLINE cTValue *lua_next_frame(cTValue *frame,
                                             bool *should_visit_frame) {
    if (frame_islua(frame)) {
        return frame_prevl(frame);
    }

    if (frame_isvarg(frame)) {
        LUA_TRACE("Skip vararg pseudo-frame.");
        *should_visit_frame = false; /* Skip vararg pseudo-frame. */
    }

    return frame_prevd(frame);
}

/**
 * @brief Resolves address of current base.
 *
 * Sometimes the first frame is not what `L->base` points to.
 * I suppose this is an optimization made in the Lua interpreter.
 * Since we don't leave the VM, we don't need to update the value.
 * All calls to extern C code that requires correct base have a code that
 * updates `L->base`.
 *
 * @param state Lua unwind state.
 * @param base Value of `L->base`.
 * @param max_stack Value of `L->maxstack`.
 * @param bottom Bottom frame of the stack.
 * @return TValue* Correct base.
 */
static ALWAYS_INLINE TValue *lua_resolve_base_value(struct lua_state *state,
                                                    TValue *base,
                                                    cTValue *max_stack,
                                                    cTValue *bottom) {
    __auto_type base_from_register = (TValue *)state->base_register;

    if ((base_from_register >= max_stack) || (bottom >= base_from_register)) {
        return base;
    }

    // TODO
    // __auto_type ins =
    // BPF_PROBE_READ_USER_POINTER((BCIns*)state->pc_register); Corner case.
    // During `BC_RET` we move results from a function to caller stack, this
    // might erase information about the last frame (which just ended). `BC_RET`
    // instruction from C functions holds information about previous frame
    // #define frame_prevl(f)		((f) - (1+LJ_FR2+bc_a(frame_pc(f)[-1])))
    // if (bc_op(ins) == BC_RET && base == base_from_register) {
    //   base = (TValue*)((char*)base_from_register + 8 * -bc_a(ins) - 0x10);
    // }

    return base_from_register;
}

/**
 * @brief
 *
 * @param state
 * @param base
 * @param g
 * @return
 */
static ALWAYS_INLINE TValue *lua_get_jit_base(struct lua_state *state,
                                              TValue *base, global_State *g) {
    __auto_type jit_base = tvref(BPF_PROBE_READ_USER(g, jit_base)) - 1;
    state->jit = false;

    if (!jit_base) {
        return base;
    }

    GCfunc *fn = frame_func(jit_base);

    if (!fn) {
        return base;
    }

    state->jit = true;
    return jit_base;
}

/**
 * @brief
 *
 * @param state
 */
static void lua_stack_walk(struct lua_state *state) {
    lua_State *L = (lua_State *)state->L;
    global_State *g = G(L);

    if (L == NULL || g == NULL) {
        metric_increment(METRIC_LUA_NULL_STATE_COUNT);
        LUA_TRACE("Invalid lua_State or global_State!");
        return;
    }

    int vmstate = BPF_PROBE_READ_USER(g, vmstate);

    if (vmstate == ~LJ_VMST_INTERP && !BPF_PROBE_READ_USER(L, cframe)) {
        LUA_TRACE("No Lua code running.");
        return;
    }

    switch (vmstate) {
    case ~LJ_VMST_INTERP:
        LUA_TRACE("Interpreter.");
        break;
    case ~LJ_VMST_C:
        LUA_TRACE("C function.");
        break;
    case ~LJ_VMST_GC:
        LUA_TRACE("Garbage collector.");
        break;
    case ~LJ_VMST_EXIT:
        LUA_TRACE("Trace exit handler.");
        break;
    case ~LJ_VMST_RECORD:
        LUA_TRACE("Trace recorder.");
        break;
    case ~LJ_VMST_OPT:
        LUA_TRACE("Optimizer.");
        break;
    case ~LJ_VMST_ASM:
        LUA_TRACE("Assembler.");
        break;
    default:
        if (vmstate >= 0) {
            LUA_TRACE("JIT Trace #%d", vmstate);
            break;
        }

        LUA_TRACE("Unknown state %d", vmstate);
        break;
    }

    cTValue *frame, *nextframe;
    __auto_type *base = BPF_PROBE_READ_USER(L, base);
    __auto_type bottom = tvref(BPF_PROBE_READ_USER(L, stack)) + LJ_FR2;
    __auto_type max_stack = tvref(BPF_PROBE_READ_USER(L, maxstack));

    // TODO: implement pc_hint for top frame
    if (vmstate == ~LJ_VMST_INTERP) {
        LUA_TRACE("Used base from register | r14 == G == %d",
                  g == ((char *)state->dispatch_register -
                        state->config.offset_g_to_dispatch));

        base = lua_resolve_base_value(state, base, max_stack, bottom);
    } else if (vmstate >= 0) {
        base = lua_get_jit_base(state, base, g);
    }

    frame = nextframe = base - 1;

    int stack_size = (max_stack - (bottom - LJ_FR2)) >> 3;
    int free_slots = (max_stack - BPF_PROBE_READ_USER(L, top) - 8) >> 3;

    LUA_TRACE("Approximate stack size: %d | L->base=%px",
              stack_size - free_slots, BPF_PROBE_READ_USER(L, base));

    int debug_count = 0;
    bool should_visit_frame = true;

    for (int i = 0; i < 10 && frame > bottom; i++) {
        if (!(frame <= max_stack && (!nextframe || nextframe <= max_stack))) {
            //LUA_TRACE("broken frame chain");
            return;
        }

        if (frame_gc(frame) == obj2gco(L)) {
            //LUA_TRACE("Skip dummy frames.");
            should_visit_frame =
                false; /* Skip dummy frames. See lj_err_optype_call(). */
        }

        /* Level found. */
        if (should_visit_frame) {
            //LUA_TRACE("level=%d found frame=%px nextframe=%px", debug_count,
            //          frame, nextframe);

            if (!lua_get_function_info(state, frame, nextframe)) {
                //LUA_TRACE("lua_get_function_info error on level %d",
                //          debug_count);
                metric_increment(METRIC_LUA_GET_FUNCTION_INFO_FAIL_COUNT);
                return;
            }

            debug_count++;
            metric_increment(METRIC_LUA_PROCESSED_FRAMES_COUNT);
        }

        should_visit_frame = true;
        nextframe = frame;
        frame = lua_next_frame(frame, &should_visit_frame);
    }
}

/**
 * @brief Performs g->cur_L using offset information from the binary.
 *
 * @param state Lua unwind state.
 * @param global_state
 * @return
 */
static ALWAYS_INLINE lua_State *
lua_global_state_get_cur_l(struct lua_state *state, void *global_state) {
    // pointer to L aka lua_State
    void *cur_L_field = (char *)global_state + state->config.offset_g_to_l;
    void *cur_L;

    long err = bpf_probe_read_user(&cur_L, sizeof(void *), cur_L_field);
    if (err != 0) {
        metric_increment(METRIC_LUA_CUR_L_READ_FAIL_COUNT);
        LUA_TRACE("get_lua_state_from_global_state: failed to read g->curL=%px "
                  "from g=%px (%ld)",
                  cur_L_field, global_state, err);
        return NULL;
    }

    return (lua_State *)cur_L;
}

/**
 * @brief
 *
 * @param state Lua unwind state.
 * @param global_state
 * @return
 */
static ALWAYS_INLINE bool lua_global_state_is_valid(struct lua_state *state,
                                                    void *global_state) {
    LUA_TRACE("lua_global_state_is_valid: Checking global_State=%px",
              global_state);

    // cross check L points to G, G points to L
    lua_State *L = lua_global_state_get_cur_l(state, global_state);

    if (!L) {
        metric_increment(METRIC_LUA_CUR_L_IS_NULL_COUNT);
        return false;
    }

    global_State *g = G(L);

    if (global_state != g) {
        metric_increment(METRIC_LUA_G_EQ_G_MISMATCH_COUNT);
        LUA_TRACE(
            "lua_global_state_is_valid: G(g->cur_L)=%px doesn't match g=%px", g,
            global_state);
        return false;
    }

    // Seems to be redunant and also misleading if compiler used different
    // offsets
    // lua_State *cur_L = gco2th(gcref(BPF_PROBE_READ_USER(g, cur_L)));

    // if (cur_L != L) {
    //     metric_increment(METRIC_LUA_L_EQ_L_MISMATCH_COUNT);
    //     LUA_TRACE("lua_global_state_is_valid: cur_L %px doesn't match L %px",
    //               cur_L, L);
    //     return false;
    // }

    LUA_TRACE("lua_global_state_is_valid: global_State=%px is valid",
              global_state);
    return true;
}

/**
 * @brief
 *
 * @param process_info Information about the current process, and specifically
 * the binary, where LuaJIT is, located.
 * @param state Lua unwind state.
 * @return
 */
static ALWAYS_INLINE bool find_lua_state(struct process_info *process_info,
                                         struct lua_state *state) {
#if 0
  READ_SAFE(globalL,
            (lua_State**)(process_info->lua_binary.base_address + 0x8b0a0),
            'G');

  // READ_SAFE(globalL,
  //           (lua_State**)(0x000077d16846bbc8),
  //           'G');

  lua_State* cur_L = gco2th(gcref(BPF_PROBE_READ_USER(G(globalL), cur_L)));

  state->L = (u64)cur_L;
  return true;
#else
    // check if we already have cached G for current process
    struct lua_global_state_key key = {.pid = state->pid};

    struct lua_global_state_cache *cached_global_state =
        bpf_map_lookup_elem(&lua_global_state_storage, &key);

    if (cached_global_state != NULL && cached_global_state->G != 0) {
        // now check, if it's still valid
        if (lua_global_state_is_valid(state, (void *)cached_global_state->G)) {
            metric_increment(METRIC_LUA_VALID_CACHE_COUNT);
            LUA_TRACE("find_lua_state: using valid global_State=%px from cache",
                      (void *)cached_global_state->G);

            state->L = (u64)lua_global_state_get_cur_l(
                state, (void *)cached_global_state->G);
            return true;
        }

        // state is no longer valid, probably called lua_Close, remove
        // cached entry
        metric_increment(METRIC_LUA_INVALIDED_CACHE_COUNT);
        LUA_TRACE("find_lua_state: remove invalid global_State=%px from cache",
                  (void *)cached_global_state->G);

        struct lua_global_state_cache entry = {};
        bpf_map_update_elem(&lua_global_state_storage, &key, &entry, BPF_ANY);
    }

    // Don't try to find the state if we are currently not executing in LuaJIT
    // binary
    if (state->instruction_pointer < process_info->lua_binary.base_address ||
        (process_info->lua_binary.base_address + state->config.binary_size) <=
            state->instruction_pointer) {
        metric_increment(METRIC_LUA_NOT_IN_LUAJIT_BINARY_COUNT);
        LUA_TRACE("find_lua_state: not in luajit binary");
        return false;
    }

    // G aka global_State
    void *global_state =
        (char *)state->dispatch_register - state->config.offset_g_to_dispatch;

    if (lua_global_state_is_valid(state, global_state)) {
        // new valid state has been found, cache it
        metric_increment(METRIC_LUA_GLOBAL_STATE_FOUND_COUNT);
        LUA_TRACE("find_lua_state: found new global_State=%px", global_state);

        struct lua_global_state_cache entry;
        entry.G = (u64)global_state;
        bpf_map_update_elem(&lua_global_state_storage, &key, &entry, BPF_ANY);

        state->L = (u64)lua_global_state_get_cur_l(state, global_state);
        return true;
    }

    metric_increment(METRIC_LUA_GLOBAL_STATE_NOT_FOUND_COUNT);
    LUA_TRACE("find_lua_state: ignore invalid global_State=%px", global_state);
    return false;
#endif
}

/**
 * @brief Entry point for collecting stack info from Lua VM.
 *
 * @param process_info Information about the current process, and specifically
 * the binary, where LuaJIT is, located.
 * @param state Lua unwind state. Not to be confused with lua_State.
 */
static ALWAYS_INLINE void lua_collect_stack(struct process_info *process_info,
                                            struct lua_state *state) {
    if (process_info == NULL || state == NULL ||
        !is_mapped(process_info->lua_binary)) {
        return;
    }

    if (!lua_state_config_get(state, process_info)) {
        return;
    }

    metric_increment(METRIC_LUA_PROCESSED_STACKS_COUNT);

    LUA_TRACE("Lua config info: offset_g_to_l=%llu offset_g_to_dispatch=%llu "
              "version=%u",
              state->config.offset_g_to_l, state->config.offset_g_to_dispatch,
              state->config.version);

    LUA_TRACE("Binary info: base_address=%llu binary_size=%llu",
              process_info->lua_binary.base_address, state->config.binary_size);

    LUA_TRACE("Compiler info: offsetof(global_State, cur_L)=%llu "
              "offsetof(lua_State, glref)=%llu",
              offsetof(global_State, cur_L), offsetof(lua_State, glref));

    if (!find_lua_state(process_info, state)) {
        return;
    }

    lua_stack_reset(state);
    lua_symbol_prepare(state);
    lua_stack_walk(state);

    LUA_TRACE("Stats: frames=%d", state->stack.len);

    return;
}
