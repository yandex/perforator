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
    u64 vm_start_pc;          // First PC of VM. Binary-relative.
    u64 vm_end_pc;            // Exclusive end of VM, including the FFI tail. Binary-relative.
    u32 ffi_call_size;        // Size of the verified FFI tail, or zero when absent.
};

BPF_MAP(lua_storage, BPF_MAP_TYPE_HASH, binary_id, struct lua_config, MAX_BINARIES);

enum lua_frame_type : u8 {
    LUA_FRAME_TYPE_LUA,
    LUA_FRAME_TYPE_C,
    LUA_FRAME_TYPE_INVALID,
    LUA_FRAME_TYPE_VM_INFO,
};

enum lua_c_frame_kind : u8 {
    LUA_C_FRAME_KIND_NATIVE,       // A native C call: closes a Lua segment when merging.
    LUA_C_FRAME_KIND_ASM_FF,       // An ASM fast function: does not split a Lua segment.
    LUA_C_FRAME_KIND_PROTECTED_FF, // pcall/xpcall identified through its callee's PCALL(H) link.
};

// Lua walk status only. Native truncation is inferred from the native stack size.
enum lua_stack_flags : u32 {
    LUA_STACK_FLAGS_LUA_FAILED = 1 << 0,
    LUA_STACK_FLAGS_LUA_TRUNCATED = 1 << 1,
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
 * `value` depends on `type`:
 *  - `LUA_FRAME_TYPE_LUA`     - Lua function. Uses `lua_frame` value.
 *  - `LUA_FRAME_TYPE_C`       - C or FastFunction. Uses `c_frame` value.
 *  - `LUA_FRAME_TYPE_INVALID` - Invalid frame. Uses `invalid_frame` value.
 *  - `LUA_FRAME_TYPE_VM_INFO` - Leading absolute VM range [start, end) and Lua walk status.
 */
struct lua_frame {
    enum lua_frame_type type;
    u8 vm_entry_before; // FRAME_CP on this callee's effective link.
    union {
        struct {
            u64 proto_address;
            i32 first_line;
            i32 current_line;
            u32 bytecode_index;
        } lua_frame;
        struct {
            u64 function_address;
            u8 ffid;
            enum lua_c_frame_kind kind;
        } c_frame;
        struct {
            u64 vm_start;
            u64 vm_end;
            u32 ffi_call_size;
            u32 flags;
        } vm_info;
        struct {
            enum lua_frame_error error;
        } invalid_frame;
    } value;
};

_Static_assert(sizeof(struct lua_frame) == 32, "Lua wire frame size changed");
_Static_assert(__builtin_offsetof(struct lua_frame, vm_entry_before) == 1, "Lua VM entry flag offset changed");
_Static_assert(__builtin_offsetof(struct lua_frame, value.c_frame.kind) == 17, "Lua C kind wire offset changed");
_Static_assert(__builtin_offsetof(struct lua_frame, value.vm_info.ffi_call_size) == 24, "Lua FFI tail size wire offset changed");
_Static_assert(__builtin_offsetof(struct lua_frame, value.vm_info.flags) == 28, "Lua header flags wire offset changed");

BTF_EXPORT(struct lua_frame);

/**
 * @brief Lua call stack.
 *
 * A nonempty wire payload contains one VM_INFO frame followed by len Lua
 * frames in leaf-to-root order. pack_sample_lang writes len + 1 records;
 * decodeLuaFrames recovers len by subtracting the header. An empty walk
 * emits no payload. During the walk, len includes the reserved slot.
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
    u64 process_starttime;    // Distinguishes PID reuse in symbol caches.
    struct lua_config config; // Config of LuaJIT binary found in this process.
    u64 binary_start_address; // Base address of LuaJIT binary in memory.
    u64 binary_end_address;   // Last address of LuaJIT binary in memory.

    // Registers
    u64 instruction_pointer;  // Value of `rip`. Used to determine if we're executing in LuaJIT binary.
    u64 dispatch_register;    // Value of `r14`. This register might hold pointer to `GG_State->dispatch`.
    u64 lua_state_register;   // Value of register for C ABI first function argument. This register might hold pointer to `lua_State`.
    u64 base_register;        // Value of `rdx`. This register might have a hint about the actual L->base value.
    u64 bytecode_pc_register; // Value of `rbx`. Inside the interpreter this register holds the bytecode PC.

    // Main structures
    u64 current_lua_state; // Current `lua_State*`. Use `lua_state_get_lua_state`.
    u64 global_state;      // Process `global_State*`. Use `lua_state_get_global_state`.
    u32 stack_flags;       // Lua walk failure/truncation, reset for every sample.

    // Stack
    struct lua_stack stack; // Call stack of current `lua_State`.
};
