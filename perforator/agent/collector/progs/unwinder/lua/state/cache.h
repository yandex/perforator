#pragma once

#include "../luajit/luajit.h"
#include "../types.h"

// namespace lua::state::cache

enum {
    // Max processes and their `global_State*` in LRU cache.
    LUA_MAX_CACHED_GLOBAL_STATES = 1024
};

/**
 * @brief Cache of `global_State*` per process.
 * Key: pid.
 * Value: `global_State*`.
 */
BPF_MAP(lua_global_state_cache, BPF_MAP_TYPE_LRU_HASH, u32, u64,
        LUA_MAX_CACHED_GLOBAL_STATES);

/**
 * @brief Get `global_State*` from cache for the specified process.
 *
 * @note To modify cached value, use `lua_state_cache_set`.
 *
 * @param state Lua unwind state.
 * @return Pointer to cached `global_State*`. May be NULL. Cached value may be
 * NULL.
 */
[[nodiscard]] static ALWAYS_INLINE global_State *const *
lua_state_cache_get(const struct lua_state *state) {
    return bpf_map_lookup_elem(&lua_global_state_cache, &state->pid);
}

/**
 * @brief Cache `global_State*` from the specified process.
 *
 * @param state Lua unwind state.
 * @return Status of `bpf_map_update_elem` operation.
 */
static ALWAYS_INLINE int lua_state_cache_set(struct lua_state *state,
                                             global_State *g) {
    return bpf_map_update_elem(&lua_global_state_cache, &state->pid, &g,
                               BPF_ANY);
}
