#pragma once

#include "../metrics.h"
#include "luajit/luajit.h"
#include "trace.h"
#include "types.h"

// namespace lua::state

enum {
    // Max processes and their `global_State*` in LRU cache.
    LUA_STATE_CACHE_SIZE = 1024
};

/**
 * @brief Cache of `global_State*` per process.
 * Key: pid.
 * Value: `global_State*`.
 */
BPF_MAP(lua_state_cache, BPF_MAP_TYPE_LRU_HASH, u32, u64, LUA_STATE_CACHE_SIZE);

/**
 * @brief Lua unwinder state.
 * Stored in `profiler_state`.
 */
struct lua_state {
    // Process info
    u32 pid;                   // Current process ID.
    struct lua_config config;  // Config of LuaJIT binary found in this process.
    u64 binary_start_address;  // Base address of LuaJIT binary in memory.
    u64 binary_end_address;    // Last address of LuaJIT binary in memory.

    // Registers
    u64 instruction_pointer;  // Value of `rip`. Used to determine if we're
                              // executing in LuaJIT binary.
    u64 dispatch_register;   // Value of `r14`. This register might hold pointer
                             // to `GG_State->dispatch`.
    u64 lua_state_register;  // Value of register for C ABI first function
                             // argument. This register might hold pointer to
                             // `lua_State`.
    u64 base_register;  // Value of `rdx`. This register might have a hint about
                        // the actual L->base value.

    // Main structures
    u64 current_lua_state;  // Current `lua_State*`. Use
                            // `lua_state_get_lua_state`.
    u64 global_state;       // Process `global_State*`. Use
                            // `lua_state_get_global_state`.

    // Stack
    struct interpreter_stack stack;  // Call stack of current `lua_State`.
};

/**
 * @brief Get and save config for the found LuaJIT binary in Lua unwind state.
 *
 * @param state Lua unwind state.
 * @param process_info Information about current process, and specifically the
 * binary where LuaJIT is located.
 * @returns `true` if config was found.
 */
[[nodiscard]] static ALWAYS_INLINE bool lua_state_get_config_from_process(
    struct lua_state* state, const struct process_info* process_info) {
    struct mapped_binary binary = process_info->lua_binary;

    struct lua_config* config = bpf_map_lookup_elem(&lua_storage, &binary.id);
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
[[nodiscard]] static ALWAYS_INLINE global_State* lua_state_get_global_state(
    struct lua_state* state) {
    return (global_State*)state->global_state;
}

/**
 * @brief Set current **valid** `global_State*` for the Lua unwind state.
 *
 * @param state Lua unwind state.
 * @param global_state Valid global state.
 */
static ALWAYS_INLINE void lua_state_set_global_state(
    struct lua_state* state, global_State* global_state) {
    state->global_state = (u64)global_state;
}

/**
 * @brief Return current **valid** `lua_State*` for the Lua unwind state.
 *
 * @param state Lua unwind state.
 */
[[nodiscard]] static ALWAYS_INLINE lua_State* lua_state_get_lua_state(
    struct lua_state* state) {
    return (lua_State*)state->current_lua_state;
}

/**
 * @brief Set current **valid** `lua_State*` for the Lua unwind state.
 *
 * @param state Lua unwind state.
 * @param L Valid Lua state.
 */
static ALWAYS_INLINE void lua_state_set_lua_state(struct lua_state* state,
                                                  lua_State* L) {
    state->current_lua_state = (u64)L;
}

/**
 * @brief Get `global_State*` from cache for the specified process.
 *
 * @note To modify cached value, use `lua_state_cache_set`.
 *
 * @param state Lua unwind state.
 * @return `global_State*` value from cache or NULL.
 */
[[nodiscard]] static ALWAYS_INLINE global_State* lua_state_cache_get(
    const struct lua_state* state) {
    global_State** global_state_holder =
        bpf_map_lookup_elem(&lua_state_cache, &state->pid);
    if (global_state_holder == NULL) {
        return NULL;
    }

    return *global_state_holder;
}

/**
 * @brief Cache `global_State*` from the specified process.
 *
 * @param state Lua unwind state.
 * @return Status of `bpf_map_update_elem` operation.
 */
[[nodiscard]] static ALWAYS_INLINE int lua_state_cache_set(
    struct lua_state* state, global_State* g) {
    return bpf_map_update_elem(&lua_state_cache, &state->pid, &g, BPF_ANY);
}

/**
 * @brief Test global state for validity and saves states to the Lua unwind
 * state.
 *
 * We check if `G(global_state->cur_L) == global_state`.
 * If they are equal, saves states.
 *
 * @param state Lua unwind state.
 * @param global_state Non-NULL but still potential state.
 * @return
 */
[[nodiscard]] static ALWAYS_INLINE bool lua_state_test_and_set(
    struct lua_state* state, global_State* global_state) {
    // cur_L must be taken using parsed offsets
    void* cur_L_field = (char*)global_state + state->config.offset_g_to_l;
    lua_State* L;

    long err = bpf_probe_read_user(&L, sizeof(lua_State*), cur_L_field);
    if (err != 0) {
        LUA_TRACE(
            "lua_state_test_and_set: failed to read global_state->curL=%px "
            "from global_state=%px (%ld)",
            cur_L_field, global_state, err);
        return false;
    }

    if (!L) {
        return false;
    }

    global_State* global_state_from_l = G(L);

    // Cross check that `global_state` points to `cur_L`, `cur_L` points to
    // `global_state`
    if (global_state != global_state_from_l) {
        LUA_TRACE(
            "lua_state_test_and_set: G(global_state->cur_L)=%px doesn't "
            "match global_state=%px",
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
[[nodiscard]] static ALWAYS_INLINE bool lua_state_find_g_and_l(
    struct lua_state* state) {
    global_State* cached_global_state = lua_state_cache_get(state);
    if (cached_global_state != NULL) {
        // Cache exists
        if (lua_state_test_and_set(state, cached_global_state)) {
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

    // Try to find state from dispatch register.
    // It's always present when we are executing inside LuaJIT VM.
    global_State* global_state =
        (global_State*)((char*)state->dispatch_register -
                        state->config.offset_g_to_dispatch);
    if (lua_state_test_and_set(state, global_state)) {
        metric_increment(METRIC_LUA_GLOBAL_STATE_FOUND_COUNT);
        LUA_TRACE("lua_state_find_g_and_l: found new global_State=%px",
                  global_state);
        (void)lua_state_cache_set(state, global_state);
        return true;
    }

    // Alternatively try to find from register for C ABI first function
    // argument.
    global_state = G((lua_State*)state->lua_state_register);
    if (lua_state_test_and_set(state, global_state)) {
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
 * Sometimes the first frame is not what `lua_state->base` points to.
 * I suppose this is an optimization made in the Lua interpreter.
 * Since we don't leave the VM, we don't need to update the value.
 * All calls to external C code that requires correct base, have a code that
 * updates `lua_state->base`.
 *
 * @param state Lua unwind state.
 * @param base Value of `lua_state->base`.
 * @param max_stack Value of `lua_state->maxstack`.
 * @param bottom Bottom frame of the stack.
 * @return TValue* Correct base.
 */
[[nodiscard]] static ALWAYS_INLINE cTValue* lua_state_resolve_base(
    struct lua_state* state, cTValue* base, cTValue* max_stack,
    cTValue* bottom) {
    u64 binary_relative_address =
        state->instruction_pointer - state->binary_start_address;
    bool is_in_luajit_vm =
        state->config.vm_start_pc <= binary_relative_address &&
        binary_relative_address < state->config.vm_end_pc;
    if (!is_in_luajit_vm) {
        metric_increment(METRIC_LUA_RDX_OUTSIDE_OF_STACK_COUNT);
        return base;
    }

    metric_increment(METRIC_LUA_BASE_DEDUCED_FROM_RDX_COUNT);
    return (cTValue*)state->base_register;
}

/**
 * @brief Resolves address of current base when executing a JIT trace.
 *
 * @param state Lua unwind state.
 * @param base Current value of base if we fail to get `jit_base`.
 * @return
 */
[[nodiscard]] static ALWAYS_INLINE cTValue* lua_state_get_jit_base(
    const struct lua_state* state, cTValue* base, global_State* g) {
    // TODO: Is it possible to have different padding? `jit_base` is next field
    // after `cur_L`. If yes, parse from binary.
    void* jit_base_pointer = (char*)g + state->config.offset_g_to_l +
                             sizeof(((global_State*)0)->cur_L);
    cTValue* jit_base =
        tvref(BPF_PROBE_READ_USER_FROM((MRef*)(jit_base_pointer)));

    if (!jit_base) {
        return base;
    }

    return jit_base;
}
