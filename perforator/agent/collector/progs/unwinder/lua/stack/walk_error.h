#pragma once

#include <bpf/bpf.h>

// namespace lua::stack

/**
 * @brief Errors during stack walk.
 */
enum lua_stack_walk_error : u8 {
    LUA_STACK_WALK_ERROR_FRAME_IS_NULL,
    LUA_STACK_WALK_ERROR_GCFUNC_IS_NULL,
    LUA_STACK_WALK_ERROR_FRAME_IS_NOT_FUNC,
    LUA_STACK_WALK_ERROR_LAST
};

BTF_EXPORT(enum lua_stack_walk_error);
