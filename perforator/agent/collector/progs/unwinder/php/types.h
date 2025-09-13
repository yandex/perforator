#pragma once

#include "../binary.h"
#include "../interpreter/types.h"

// https://github.com/php/php-src/blob/4d9fc506df1131c630c530a0bfa6d0338cffa03c/Zend/zend_compile.h#L641
// identify last frame
#define ZEND_TOP_FUNCTION      0b100000000000000000 // 1 << 17

// https://github.com/php/php-src/blob/4d9fc506df1131c630c530a0bfa6d0338cffa03c/Zend/zend_compile.h#L1065
#define ZEND_USER_FUNCTION     2
#define ZEND_EVAL_CODE         4

// https://github.com/php/php-src/blob/c40dcf93d0b95d9a1026476b202f5aa965fb3fdd/Zend/zend_compile.h#L503
struct php_execute_data_offsets {
    u32 function;
    // https://github.com/php/php-src/blob/c40dcf93d0b95d9a1026476b202f5aa965fb3fdd/Zend/zend_compile.h#L554
    // https://github.com/php/php-src/blob/c40dcf93d0b95d9a1026476b202f5aa965fb3fdd/Zend/zend_compile.h#L508
    u32 this_type_info;
    u32 prev_execute_data;
};

// https://github.com/php/php-src/blob/c40dcf93d0b95d9a1026476b202f5aa965fb3fdd/Zend/zend_compile.h#L414
struct php_op_array_offsets {
    u32 filename;
    u32 linestart;
};

// https://github.com/php/php-src/blob/c40dcf93d0b95d9a1026476b202f5aa965fb3fdd/Zend/zend_compile.h#L483
struct php_function_offsets {
    u32 type;
    u32 common_funcname;
    struct php_op_array_offsets op_array;
};

// https://github.com/php/php-src/blob/f11ea2ae1393e09bd343f8a714b7a0d9b22e1054/Zend/zend_types.h#L393
struct php_zend_string_offsets {
    u32 val;
    u32 len;
};

struct php_internals_offsets {
    u32 zend_execute_data;
    struct php_execute_data_offsets execute_data;
    struct php_function_offsets function;
    struct php_zend_string_offsets zend_string;
};

struct php_config {
    u32 version;
    u64 executor_globals_elf_vaddr;

    struct php_internals_offsets offsets;
};

enum {
    PHP_MAX_STACK_DEPTH = 128,
};

#define PHP_FRAME_UNKNOWN (struct interpreter_frame){.symbol_key = {.object_addr = 0, .pid = 0, .linestart = 0}}


struct php_function_data {
    u64 filename_addr;
    u64 funcname_addr;
};

struct php_state {
    struct php_config config;

    u64 executor_globals_mem_vaddr;
    u64 execute_data;

    u8 frame_count;
    struct interpreter_frame frames[PHP_MAX_STACK_DEPTH];

    struct symbol symbol;
    struct symbol_key symbol_key;
    struct php_function_data function_data;
    u32 pid;
};

#ifdef PERFORATOR_ENABLE_PHP

enum {
    MAX_PHP_BINARIES = MAX_BINARIES,
};

#else

enum {
    MAX_PHP_BINARIES = 1,
};

#endif

BPF_MAP(php_storage, BPF_MAP_TYPE_HASH, binary_id, struct php_config, MAX_PHP_BINARIES);
