#pragma once

#include "../binary.h"
#include "../interpreter/types.h"
#include "../output.h"

enum {
    // Maximum stack size to walk.
    LUA_MAX_STACK_DEPTH = COMMON_MAX_STACK_DEPTH,
    // Verifier mask for stack length
    LUA_MAX_STACK_DEPTH_VERIFIER_MASK = LUA_MAX_STACK_DEPTH - 1,
};

/**
 * @brief Lua config for the binary.
 * Filled from BinaryAnalysis.
 */
struct lua_config {
    u64 offset_g_to_l;         // `offsetof(global_State, cur_L)`
    u64 offset_g_to_dispatch;  // `GG_G2DISP`
    u64 binary_size;  // Size of LuaJIT binary. Used to determine if current
                      // `rip` is from this binary.
    u64 vm_start_pc;  // First PC of VM. Relative to the binary!
    u64 vm_end_pc;    // Last PC of VM. Relative to the binary!
};

BPF_MAP(lua_storage, BPF_MAP_TYPE_HASH, binary_id, struct lua_config,
        MAX_BINARIES);
