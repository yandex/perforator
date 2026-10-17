#pragma once

#include <bpf/bpf.h>

#define LUA_TRACE_LEVEL(level, fmt, ...) BPF_TRACE("lua: " level " " fmt "\n", ##__VA_ARGS__)

#define LUA_LOG_FATAL(fmt, ...) LUA_TRACE_LEVEL("[FATAL]", fmt, ##__VA_ARGS__)
#define LUA_LOG_ERROR(fmt, ...) LUA_TRACE_LEVEL("[ERROR]", fmt, ##__VA_ARGS__)
#define LUA_LOG_INFO(fmt, ...) LUA_TRACE_LEVEL("[INFO ]", fmt, ##__VA_ARGS__)
#define LUA_LOG_DEBUG(fmt, ...) LUA_TRACE_LEVEL("[DEBUG]", fmt, ##__VA_ARGS__)

// Turns on assertions for development purposes.
#ifdef LUA_ENABLE_DEV_ASSERTS

// Sends SIGSTOP (19) if assertion fails. Attach the debugger afterwards.
#define LUA_DEV_ASSERT(expression, fmt, ...) \
    ({ \
        if (!(expression)) { \
            LUA_LOG_FATAL("Assertion failed: " fmt, ##__VA_ARGS__); \
            bpf_send_signal(19); \
        } \
    })

#else // ^^^ LUA_ENABLE_DEV_ASSERTS vvv !LUA_ENABLE_DEV_ASSERTS

#define LUA_DEV_ASSERT(condition, message)

#endif // ^^^ !LUA_ENABLE_DEV_ASSERTS
