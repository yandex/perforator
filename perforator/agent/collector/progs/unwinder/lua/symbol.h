#pragma once

#include "types.h"

// namespace lua::symbol

/**
 * @brief Writes string to symbol buffer.
 *
 * @note The size of the string must be known at compile-time.
 * @see `LUA_SYMBOL_APPEND_LITERAL`
 *
 * @param caret Pointer from symbol where to append.
 * @param string String to write.
 * @param string_size Size of the string.
 * @return u64 Size written.
 */
[[nodiscard]] static ALWAYS_INLINE u64
lua_symbol_append_string(char* caret, const char* string, u64 string_size) {
    memcpy(caret, string, string_size);

    return string_size;
}

/**
 * @brief Writes literal or char array to symbol buffer.
 *
 * @see `lua_symbol_append_string`
 * @note Writes without a NULL terminator.
 *
 * @param caret Pointer from symbol where to append.
 * @param literal Literal to write
 * @return u64 Size written.
 */
#define LUA_SYMBOL_APPEND_LITERAL(caret, literal) \
    lua_symbol_append_string((caret), (literal), sizeof(literal) - 1)

/**
 * @brief Writes error message to symbol buffer
 * @note Writes without a NULL terminator.
 *
 * @param caret Pointer from symbol where to append.
 * @return u64 Size written.
 */
[[nodiscard]] static ALWAYS_INLINE u64 lua_symbol_append_fail(char* caret) {
    const char kInvalidString[] = "<failed to read>";

    return LUA_SYMBOL_APPEND_LITERAL(caret, kInvalidString);
}
