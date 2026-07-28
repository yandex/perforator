#pragma once

#include "../binary.h"
#include "../interpreter/types.h"
#include "../output.h"

enum {
    // Maximum stack size to walk.
    LUA_MAX_STACK_DEPTH = PYTHON_MAX_STACK_DEPTH,
};

/**
 * @brief Lua config for the binary.
 * Filled from BinaryAnalysis.
 */
struct lua_config {
    u32 version; // Version of LuaJIT. Encoded as (minor << 8) + (major << 16).
                 // See encodeVersion@lua.go
    u64 offset_g_to_l;        // `offsetof(global_State, cur_L)`
    u64 offset_g_to_dispatch; // `GG_G2DISP`
    u64 binary_size; // Size of LuaJIT binary. Used to determine if current
                     // `rip` is from this binary.
    u64 vm_start_pc; // First PC of VM. Relative to the binary!
    u64 vm_end_pc;   // Last PC of VM. Relative to the binary!
};

BPF_MAP(lua_storage, BPF_MAP_TYPE_HASH, binary_id, struct lua_config,
        MAX_BINARIES);

/**
 * @brief Lua unwinder state.
 * Stored in `profiler_state`.
 */
struct lua_state {
    // Process info
    u32 pid;                  // Current process ID.
    struct lua_config config; // Config of LuaJIT binary found in this process.
    u64 binary_start_address; // Base address of LuaJIT binary in memory.
    u64 binary_end_address;   // Last address of LuaJIT binary in memory.

    // Registers
    u64 instruction_pointer; // Value of `rip`. Used to determine if we're
                             // executing in LuaJIT binary.
    u64 dispatch_register;   // Value of `r14`. This register might hold pointer
                             // to `GG_State->dispatch`.
    u64 l_register; // Value of `rdi`. This register might hold pointer to `L`.
    u64 base_register; // Value of `rdx`. This register might have a hint about
                       // the actual L->base value.

    // Main structures
    u64 L; // Current `lua_State*`. Use `lua_state_get_lua_state`.
    u64 g; // Process `global_State*`. Use `lua_state_get_global_state`.

    // Stack
    struct interpreter_stack stack; // Call stack of current `lua_State`.
};
