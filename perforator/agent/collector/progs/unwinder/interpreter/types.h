#pragma once

struct symbol_key {
    u64 object_addr;
    u32 pid;
    i32 linestart;
};

enum {
    // This constant should be a power of 2.
    SYMBOL_BUFFER_SIZE = 1 << 10,
    // This constant is intentionally PYTHON_SYMBOL_BUFFER_SIZE - 1,
    // we use it for ending length to satisfy the BPF verifier.
    INTERPRETER_SYMBOL_STRING_LENGTH_VERIFIER_MASK = SYMBOL_BUFFER_SIZE - 1,
    MAX_SYMBOLS_SIZE = 200000,
};

struct symbol {
    // Both lengths are in codepoints.
    u8 name_length;
    u8 filename_length;
    u8 codepoint_size; // 1 for ascii, 2 for ucs2, 4 for ucs4
    // The layout is [name][filename].
    // We can store expensive ucs4 encoded strings here for legacy CPython.
    char data[SYMBOL_BUFFER_SIZE];
};

struct interpreter_frame {
    struct symbol_key symbol_key;
};

BPF_MAP(interpreter_symbols, BPF_MAP_TYPE_LRU_HASH, struct symbol_key, struct symbol, MAX_SYMBOLS_SIZE);
