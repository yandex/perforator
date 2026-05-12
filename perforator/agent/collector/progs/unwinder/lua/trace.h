#pragma once

#define LUA_TRACE(fmt, ...) \
    BPF_TRACE("lua: " fmt "\n", ##__VA_ARGS__)
