#pragma once

// Temporary change to get rid of bad address error when running without --debug flag
#define LUA_TRACE(fmt, ...) \
    BPF_PRINTK("lua: " fmt "\n", ##__VA_ARGS__)
