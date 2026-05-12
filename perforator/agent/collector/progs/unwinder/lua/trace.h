#pragma once

// Temporary change to get rid of bad address error when running without --debug flag
#define LUA_TRACE(fmt, ...) \
    BPF_TRACE("lua: " fmt "\n", ##__VA_ARGS__)
