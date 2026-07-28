#pragma once

#include "../../metrics.h"
#include "../trace.h"
#include "../types.h"
#include "cache.h"

// namespace lua::state

/**
 * @brief Get and save config for the found LuaJIT binary in Lua unwind state.
 *
 * @param state Lua unwind state.
 * @param process_info Information about current process, and specifically the
 * binary where LuaJIT is located.
 * @returns `true` if config was found.
 */
[[nodiscard]] static ALWAYS_INLINE bool
lua_state_get_config_from_process(struct lua_state *state,
                                  const struct process_info *process_info) {
    __auto_type binary = process_info->lua_binary;

    struct lua_config *config = bpf_map_lookup_elem(&lua_storage, &binary.id);
    if (config == NULL) {
        return false;
    }

    state->config = *config;
    state->binary_start_address = binary.base_address;
    state->binary_end_address = binary.base_address + config->binary_size;

    return true;
}

/**
 * @brief Return current **valid** `global_State*` for the Lua unwind state.
 *
 * @param state Lua unwind state.
 *
 * @warning Can be used only after lua_state_find_g_and_l().
 */
[[nodiscard]] static ALWAYS_INLINE global_State *
lua_state_get_global_state(struct lua_state *state) {
    return (global_State *)state->g;
}

/**
 * @brief Set current **valid** `global_State*` for the Lua unwind state.
 *
 * @param state Lua unwind state.
 * @param global_state Valid global state.
 */
static ALWAYS_INLINE void
lua_state_set_global_state(struct lua_state *state,
                           global_State *global_state) {
    state->g = (u64)global_state;
}

/**
 * @brief Return current **valid** `lua_State*` for the Lua unwind state.
 *
 * @param state Lua unwind state.
 */
[[nodiscard]] static ALWAYS_INLINE lua_State *
lua_state_get_lua_state(struct lua_state *state) {
    return (lua_State *)state->L;
}

/**
 * @brief Set current **valid** `lua_State*` for the Lua unwind state.
 *
 * @param state Lua unwind state.
 * @param L Valid Lua state.
 */
static ALWAYS_INLINE void lua_state_set_lua_state(struct lua_state *state,
                                                  lua_State *L) {
    state->L = (u64)L;
}

/**
 * @brief Test global state for validity and saves G and L to the Lua unwind
 * state.
 *
 * We check if `G(global_state->cur_L) == global_state`.
 * If they are equal, saves G and L.
 *
 * @param state Lua unwind state.
 * @param global_state Non-NULL but still potential state.
 * @return
 */
[[nodiscard]] static ALWAYS_INLINE bool
lua_state_test_and_set(struct lua_state *state, global_State *global_state) {
    // cur_L must be taken using parsed offsets
    void *cur_L_field = (char *)global_state + state->config.offset_g_to_l;
    lua_State *L;

    long err = bpf_probe_read_user(&L, sizeof(lua_State *), cur_L_field);
    if (err != 0) {
        LUA_TRACE("lua_state_test_and_set: failed to read g->curL=%px "
                  "from g=%px (%ld)",
                  cur_L_field, global_state, err);
        return false;
    }

    if (!L) {
        return false;
    }

    global_State *global_state_from_l = G(L);

    // Cross check that G points to L, L points to G
    if (global_state != global_state_from_l) {
        LUA_TRACE("lua_state_test_and_set: G(g->cur_L)=%px doesn't match g=%px",
                  global_state_from_l, global_state);
        return false;
    }

    lua_state_set_global_state(state, global_state);
    lua_state_set_lua_state(state, L);

    return true;
}

/**
 * @brief Find and save process `global_State` (`G`) and current `lua_State`
 * (`L`).
 *
 * This function caches found G and reuses it if it's valid. If no cache exists,
 * function checks if current `rip` is inside LuaJIT binary to minimize CPU
 * cost. When in LuaJIT binary, function will try to find `G` from dispatch
 * register (`r14`).
 *
 * @param process_info Information about the current process, and specifically
 * the binary, where LuaJIT is, located.
 * @param state Lua unwind state.
 * @return `true` if G and L were found.
 */
[[nodiscard]] static ALWAYS_INLINE bool
lua_state_find_g_and_l(struct lua_state *state) {
    global_State *const *cached_global_state = lua_state_cache_get(state);

    if (cached_global_state != NULL && *cached_global_state != NULL) {
        // Cache exists
        if (lua_state_test_and_set(state, *cached_global_state)) {
            metric_increment(METRIC_LUA_VALID_CACHE_COUNT);
            // L and G are set in `lua_state_test_and_set`
            return true;
        }

        // The state is no longer valid, probably called lua_close
        // Erase cached pointer
        metric_increment(METRIC_LUA_INVALIDED_CACHE_COUNT);
        (void)lua_state_cache_set(state, NULL);
    }

    // Cache doesn't exist or invalidated

    bool is_in_luajit_binary =
        state->binary_start_address <= state->instruction_pointer &&
        state->instruction_pointer < state->binary_end_address;

    // Don't try to find the state if we are currently not executing in LuaJIT
    // binary
    if (!is_in_luajit_binary) {
        metric_increment(METRIC_LUA_NOT_IN_LUAJIT_BINARY_COUNT);
        LUA_TRACE(
            "lua_state_find_g_and_l: not in luajit binary %px <= %px < %px",
            state->binary_start_address, state->instruction_pointer,
            state->binary_end_address);
        return false;
    }

    global_State *global_state;

    // TODO: Alternatively we can probe both register and save that was valid
    __auto_type binary_relative_address =
        state->instruction_pointer - state->binary_start_address;
    bool is_in_luajit_vm =
        state->config.vm_start_pc <= binary_relative_address &&
        binary_relative_address < state->config.vm_end_pc;
    if (is_in_luajit_vm) {
        // Inside VM the dispatch register will always hold dispatch field.
        // Subtracting `GG_G2DISP` will get `global_State*`.
        global_state = (global_State *)((char *)state->dispatch_register -
                                        state->config.offset_g_to_dispatch);
        LUA_TRACE("Inside VM");
    } else {
        // Outside VM we might have `lua_State*` as the first argument.
        // We can't assume it's `cur_L`, so we need to get back to
        // `global_State*` and get actual `cur_L` from there.
        global_state = G((lua_State *)state->l_register);
        LUA_TRACE("Outside VM");
    }

    if (lua_state_test_and_set(state, global_state)) {
        // L and G are set in `lua_state_test_and_set`
        metric_increment(METRIC_LUA_GLOBAL_STATE_FOUND_COUNT);
        LUA_TRACE("lua_state_find_g_and_l: found new global_State=%px",
                  global_state);
        (void)lua_state_cache_set(state, global_state);
        return true;
    }

    metric_increment(METRIC_LUA_GLOBAL_STATE_NOT_FOUND_COUNT);
    LUA_TRACE("lua_state_find_g_and_l: ignore invalid global_State=%px",
              global_state);
    return false;
}

/**
 * @brief Resolves address of current base when in interpreter.
 *
 * Sometimes the first frame is not what `L->base` points to.
 * I suppose this is an optimization made in the Lua interpreter.
 * Since we don't leave the VM, we don't need to update the value.
 * All calls to external C code that requires correct base, have a code that
 * updates `L->base`.
 *
 * @param state Lua unwind state.
 * @param base Value of `L->base`.
 * @param max_stack Value of `L->maxstack`.
 * @param bottom Bottom frame of the stack.
 * @return TValue* Correct base.
 */
[[nodiscard]] static ALWAYS_INLINE cTValue *
lua_state_resolve_base(struct lua_state *state, cTValue *base,
                       cTValue *max_stack, cTValue *bottom) {
    __auto_type base_from_register = (cTValue *)state->base_register;

    __auto_type binary_relative_address =
    state->instruction_pointer - state->binary_start_address;
    bool is_in_luajit_vm =
        state->config.vm_start_pc <= binary_relative_address &&
        binary_relative_address < state->config.vm_end_pc;
    if (!is_in_luajit_vm) {
        metric_increment(METRIC_LUA_RDX_OUTSIDE_OF_STACK_COUNT);
        return base;
    }

    __auto_type ins = BPF_PROBE_READ_USER_FROM((BCIns *)state->pc_register);

    // During `BC_RET` we move results from a function to caller stack, this
    // might erase information about the last frame (which just ended).
    // `BC_RET` instruction from C functions holds information about previous
    // frame. TODO: Can it be improved?
    if (bc_op(ins) == BC_RET && base == base_from_register) {
        metric_increment(METRIC_LUA_BASE_DEDUCED_FROM_RET_COUNT);
        return (TValue *)((char *)base_from_register + 8 * -bc_a(ins) - 0x10);
    }

    metric_increment(METRIC_LUA_BASE_DEDUCED_FROM_RDX_COUNT);
    return base_from_register;
}

/**
 * @brief Resolves address of current base when executing a JIT trace.
 *
 * @param state Lua unwind state.
 * @param base Current value of base if we fail to get `jit_base`.
 * @return
 */
[[nodiscard]] static ALWAYS_INLINE cTValue *
lua_state_get_jit_base(const struct lua_state *state, cTValue *base,
                       global_State *g) {
    // TODO: Is it possible to have different padding? `jit_base` is next field
    // after `cur_L`. If yes, parse from binary.
    __auto_type jit_base_pointer = (char *)g + state->config.offset_g_to_l +
                                   sizeof(((global_State *)0)->cur_L);
    __auto_type jit_base =
        tvref(BPF_PROBE_READ_USER_FROM((MRef *)(jit_base_pointer)));

    if (!jit_base) {
        return base;
    }

    return jit_base;
}
