#pragma once

#include "../binary.h"
#include "../interpreter/types.h"

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
    u64 offset_g_to_l;        // `offsetof(global_State, cur_L)`
    u64 offset_g_to_dispatch; // `GG_G2DISP`
    u64 offset_g_to_vm_state; // `offsetof(global_State, vmstate)`
    u64 binary_size;          // Size of LuaJIT binary. Used to determine if current `rip` is from this binary.
    u64 vm_start_pc;          // First PC of VM. Relative to the binary!
    u64 vm_end_pc;            // Last PC of VM. Relative to the binary!
};

BPF_MAP(lua_storage, BPF_MAP_TYPE_HASH, binary_id, struct lua_config, MAX_BINARIES);

enum lua_frame_type : u8 {
    LUA_FRAME_TYPE_LUA,
    LUA_FRAME_TYPE_C,
    LUA_FRAME_TYPE_INVALID,
};

/**
 * @brief Invalid frame errors
 */
enum lua_frame_error : u8 {
    LUA_FRAME_ERROR_GCFUNC_IS_NULL,
    LUA_FRAME_ERROR_GCFUNC_WRONG_TYPE,
};

/**
 * @brief Frame in the Lua stack.
 *
 * `value` can be one of three types, depending on `type`:
 *  - `LUA_FRAME_TYPE_LUA`     - Lua function. Uses `lua_frame` value.
 *  - `LUA_FRAME_TYPE_C`       - C or FastFunction. Uses `c_frame` value.
 *  - `LUA_FRAME_TYPE_INVALID` - Invalid frame. Uses `invalid_frame` value.
 */
struct lua_frame {
    enum lua_frame_type type;
    union {
        struct symbol_key lua_frame;
        struct {
            u64 object_addr;
            u8 ffid;
        } c_frame;
        struct {
            enum lua_frame_error error;
        } invalid_frame;
    } value;
};

BTF_EXPORT(struct lua_frame);

/**
 * @brief Lua call stack.
 *
 * Array of `lua_frame`s with current length.
 */
struct lua_stack {
    struct lua_frame frames[COMMON_MAX_STACK_DEPTH];
    u8 len;
};

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
    u64 instruction_pointer; // Value of `rip`. Used to determine if we're executing in LuaJIT binary.
    u64 dispatch_register;   // Value of `r14`. This register might hold pointer to `GG_State->dispatch`.
    u64 lua_state_register;  // Value of register for C ABI first function argument. This register might hold pointer to `lua_State`.
    u64 base_register;       // Value of `rdx`. This register might have a hint about the actual L->base value.

    // Main structures
    u64 current_lua_state; // Current `lua_State*`. Use `lua_state_get_lua_state`.
    u64 global_state;      // Process `global_State*`. Use `lua_state_get_global_state`.

    // Stack
    struct lua_stack stack; // Call stack of current `lua_State`.
};
