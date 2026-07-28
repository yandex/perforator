#pragma once

#include <bpf/bpf.h>

#define LUA_TRACE(fmt, ...) BPF_TRACE("lua: " fmt "\n", ##__VA_ARGS__)
