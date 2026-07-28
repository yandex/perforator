#pragma once

#include <bpf/bpf.h>

#include "../luajit.h"
#include "../stack/walk_error.h"

// namespace lua::frame::pack

/**
 * Packed layout for frames.
 *
 * `struct symbol_key` has limited space and we need to fit Lua, C, FF and invalid frames into it!
 *
 *  - Lua frame
 *
 *                 object_addr                          linestart
 *   63...............48 47..................0    31..................0
 *  +-------------------+---------------------+  +---------------------+
 *  |   reserved        |      proto_ptr      |  |      firstline      |
 *  +-------------------+---------------------+  +---------------------+
 *
 *  - C/FF frame
 *
 *                 object_addr                          linestart
 *   63.56 55.........48 47..................0    31..................0
 *  +-----+-------------+---------------------+  +---------------------+
 *  |  0  |    ffid     |       func_ptr      |  |         -1          |
 *  +-----+-------------+---------------------+  +---------------------+
 *
 *  - Invalid frame
 *
 *                 object_addr                          linestart
 *   63.58 57.49 48...48 47..................0    31..................0
 *  +-----+-----+-------+---------------------+  +---------------------+
 *  |  0  | gct | error |          0          |  |          0          |
 *  +-----+-----+-------+---------------------+  +---------------------+
 */

enum {
    // In canonical form in most architectures userspace addresses are 48 bits long.
    // This allows to store additional data in upper 16 bits.
    LUA_OBJECT_ADDRESS_PTR_BITS = 48,
    // Mask for canonical pointer address.
    LUA_OBJECT_ADDRESS_MASK = (1ULL << LUA_OBJECT_ADDRESS_PTR_BITS) - 1,

    // Offset of ffid in c/ff frame.
    LUA_OBJECT_ADDRESS_FFID_OFFSET = LUA_OBJECT_ADDRESS_PTR_BITS,

    // Offset for stack walk error code in invalid frame.
    LUA_OBJECT_ADDRESS_STACK_WALK_ERROR_OFFSET = LUA_OBJECT_ADDRESS_PTR_BITS,
    // Bits taken by stack walk error code.
    LUA_OBJECT_ADDRESS_STACK_WALK_ERROR_BITS = 1,

    // Offset for gct in invalid frame.
    LUA_OBJECT_ADDRESS_GCT_OFFSET = LUA_OBJECT_ADDRESS_STACK_WALK_ERROR_OFFSET + LUA_OBJECT_ADDRESS_STACK_WALK_ERROR_BITS,

    // Value for non-Lua frame. `LJ_MAX_LINE` is 0x7fffff00.
    LUA_LINESTART_NON_LUA_FRAME = (u32)-1
};

_Static_assert(LUA_STACK_WALK_ERROR_LAST <= (1 << LUA_OBJECT_ADDRESS_STACK_WALK_ERROR_BITS), "lua_stack_walk_error bits mismatch");

/**
 * @brief Packs Lua frame info for `object_addr` field.
 *
 * @param proto Frame function.
 * @return Value ready to be used for `object_addr` field.
 */
[[nodiscard]] static ALWAYS_INLINE u64 lua_frame_pack_lua_object_address(luajit_gc_proto* proto) {
    return (u64)proto;
}

/**
 * @brief Packs C or FF frame info for `object_addr` field.
 *
 * @param c_function Lua C function.
 * @param ffid Fast function ID.
 * @return Value ready to be used for `object_addr` field.
 */
[[nodiscard]] static ALWAYS_INLINE u64 lua_frame_pack_c_object_address(void* c_function, u8 ffid) {
    u64 object_address = ((u64)c_function & LUA_OBJECT_ADDRESS_MASK);
    object_address |= ((u64)ffid << LUA_OBJECT_ADDRESS_FFID_OFFSET);

    return object_address;
}

/**
 * @brief Packs invalid frame info for `object_addr` field.
 *
 * @param error Error during stack walking.
 * @return Value ready to be used for `object_addr` field.
 */
[[nodiscard]] static ALWAYS_INLINE u64 lua_frame_pack_invalid_object_address(enum lua_stack_walk_error error, u8 gct) {
    u64 object_address = 0;
    object_address |= ((u64)error << LUA_OBJECT_ADDRESS_STACK_WALK_ERROR_OFFSET);
    object_address |= ((u64)gct << LUA_OBJECT_ADDRESS_GCT_OFFSET);

    return object_address;
}
