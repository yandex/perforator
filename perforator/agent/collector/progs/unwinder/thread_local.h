#pragma once

#include "binary.h"
#include "core.h"
#include "process.h"

#include <bpf/bpf.h>

#include <perforator/lib/tls/magic_bytes.h>

enum {
    MAX_TRACKED_THREAD_LOCALS_PER_BINARY = 4,
    THREAD_LOCAL_TYPE_ENUM_OFFSET = 7,
    THREAD_LOCAL_MAGIC_BYTES = 8,

    MAX_THREAD_LOCAL_STRING_LENGTH = 128
};

enum tls_variable_type : u8 {
    THREAD_LOCAL_UINT64_TYPE = 1,
    THREAD_LOCAL_STRING_TYPE = 2,
};

struct tls_binary_config {
    u64 offsets[MAX_TRACKED_THREAD_LOCALS_PER_BINARY];
};

BPF_MAP(tls_storage, BPF_MAP_TYPE_HASH, binary_id, struct tls_binary_config, MAX_BINARIES);

struct thread_local_string {
    u64 len;
    char string[MAX_THREAD_LOCAL_STRING_LENGTH];
};

union thread_local_variable {
    struct thread_local_string string;
    u64 number;
};

struct thread_local_variable_collect_result {
    u64 offset;
    enum tls_variable_type type;
    union thread_local_variable value;
};

struct tls_collect_result {
    struct thread_local_variable_collect_result values[MAX_TRACKED_THREAD_LOCALS_PER_BINARY];
};

static ALWAYS_INLINE bool collect_tls_ui64(void* uaddr, struct thread_local_variable_collect_result* result) {
    long err = bpf_probe_read_user(&result->value.number, sizeof(u64), uaddr);
    if (err != 0) {
        BPF_TRACE("failed to read thread local u64 from user space memory %p: %d", uaddr, err);
        return false;
    }

    BPF_TRACE("tls: successfully collected tls ui64 from addr %p", uaddr);

    return true;
}

static ALWAYS_INLINE bool collect_tls_string(void* uaddr, struct thread_local_variable_collect_result* result) {
    u64 strPtr = 0;
    u64 strSize = 0;
    long err = bpf_probe_read_user(&strPtr, sizeof(strPtr), uaddr);
    if (err != 0) {
        BPF_TRACE("failed to read thread local str pointer from user space memory %p: %d", uaddr, err);
        return false;
    }
    err = bpf_probe_read_user(&strSize, sizeof(strSize), uaddr + sizeof(strPtr));
    if (err != 0) {
        BPF_TRACE("failed to read thread local str size from user space memory %p: %d", uaddr, err);
        return false;
    }

    if (strPtr != 0 && strSize != 0) {
        if (strSize > MAX_THREAD_LOCAL_STRING_LENGTH) {
            strSize = MAX_THREAD_LOCAL_STRING_LENGTH;
        }

        err = bpf_probe_read_user(&result->value.string.string, strSize, (void*) strPtr);
        if (err < 0) {
            BPF_TRACE("failed to read thread local string from user space memory %p: %d", (void*)strPtr, err);
            return false;
        }
    }
    result->value.string.len = strSize;

    BPF_TRACE("tls: successfully collected tls string from addr %p", uaddr);

    return true;
}

static ALWAYS_INLINE bool check_magic_bytes(u8* magic_bytes) {
    return magic_bytes[0] == PERFORATOR_TLS_MAGIC_BYTE_0
        && magic_bytes[1] == PERFORATOR_TLS_MAGIC_BYTE_1
        && magic_bytes[2] == PERFORATOR_TLS_MAGIC_BYTE_2
        && magic_bytes[3] == PERFORATOR_TLS_MAGIC_BYTE_3
        && magic_bytes[4] == PERFORATOR_TLS_MAGIC_BYTE_4
        && magic_bytes[5] == PERFORATOR_TLS_MAGIC_BYTE_5
        && magic_bytes[6] == PERFORATOR_TLS_MAGIC_BYTE_6;
}

static ALWAYS_INLINE bool collect_tls_value(
    u64 uthread,
    u64 offset,
    struct thread_local_variable_collect_result* result
) {
    if (result == NULL) {
        return false;
    }

    void* variable = (void*) (uthread - offset);
    u8 magic_bytes[THREAD_LOCAL_MAGIC_BYTES];

    long err = bpf_probe_read_user(magic_bytes, THREAD_LOCAL_MAGIC_BYTES, variable);
    if (err != 0) {
        BPF_TRACE("failed to read thread local magic from user space addr %p: %d", variable, err);
    }

    if (!check_magic_bytes(magic_bytes)) {
        BPF_TRACE("magic bytes are not valid, uaddr: %p", variable);
        return false;
    }

    variable += THREAD_LOCAL_MAGIC_BYTES;

    bool collected = false;
    switch (magic_bytes[THREAD_LOCAL_TYPE_ENUM_OFFSET]) {
    case THREAD_LOCAL_UINT64_TYPE:
        result->type = THREAD_LOCAL_UINT64_TYPE;
        collected = collect_tls_ui64(variable, result);
        break;
    case THREAD_LOCAL_STRING_TYPE:
        result->type = THREAD_LOCAL_STRING_TYPE;
        collected = collect_tls_string(variable, result);
        break;
    default:
        BPF_TRACE("unsupported type enum value: %d", magic_bytes[THREAD_LOCAL_TYPE_ENUM_OFFSET]);
        return false;
    }

    if (collected) {
        result->offset = offset;
    }

    return collected;
}

static ALWAYS_INLINE void collect_tls_values(struct process_info* proc_info, struct tls_collect_result* result) {
    if (proc_info == NULL || result == NULL) {
        return;
    }

    binary_id id = proc_info->main_binary_id;
    struct tls_binary_config* tls_config = bpf_map_lookup_elem(&tls_storage, &id);
    if (tls_config == NULL) {
        return;
    }

    struct task_struct* task = (void*)bpf_get_current_task();

    unsigned long fsbase = BPF_CORE_READ(task, thread.fsbase);
    u64 uthread = (u64) fsbase;

    BPF_TRACE("tls: read fsbase %p", fsbase);

    int resultIndex = 0;
    for (int i = 0; i < MAX_TRACKED_THREAD_LOCALS_PER_BINARY; ++i) {
        if (tls_config->offsets[i] == 0) {
            result->values[resultIndex].offset = 0;
            break;
        }

        bool collected = collect_tls_value(
            uthread,
            tls_config->offsets[i],
            &result->values[resultIndex]
        );

        resultIndex += collected;
    }

    for (int i = resultIndex; i < MAX_TRACKED_THREAD_LOCALS_PER_BINARY; ++i) {
        result->values[i].offset = 0;
    }
}
