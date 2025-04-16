#pragma once

#include "binary.h"
#include "thread.h"

#include <bpf/bpf.h>
#include <bpf/types.h>

struct pthread_key_data {
    u64 size;
    u64 value_offset;
    u64 seq_offset;
};

struct pthread_config {
    struct pthread_key_data key_data;
    u64 first_specific_block_offset;
    u64 specific_array_offset;
    u64 struct_pthread_pointer_offset;
    u64 key_second_level_size;
    u64 key_first_level_size;
    u64 keys_max;
};

BPF_MAP(pthread_storage, BPF_MAP_TYPE_HASH, binary_id, struct pthread_config, MAX_BINARIES);

static ALWAYS_INLINE void* pthread_read_tss_first_level(void* block, u32 index) {
    if (block == NULL) {
        return NULL;
    }

    void *result = NULL;
    long err = bpf_probe_read_user(&result, sizeof(void*), (void*) (block + index * sizeof(void*)));
    if (err != 0) {
        BPF_TRACE("pthread: failed to read second level block %p: %d", block + index * sizeof(void*), err);
        return NULL;
    }

    return result;
}

static ALWAYS_INLINE void* pthread_read_tss_second_level(struct pthread_key_data* key_data_config, void* block, u32 index) {
    if (block == NULL) {
        return NULL;
    }

    void *result = NULL;
    long err = bpf_probe_read_user(&result, sizeof(void*), (void*) (block + index * key_data_config->size + key_data_config->value_offset));
    if (err != 0) {
        BPF_TRACE("pthread: failed to read value %p: %d", block + index * key_data_config->size + key_data_config->value_offset, err);
        return NULL;
    }

    BPF_TRACE("pthread: read value %p", result);

    return result;
}

static NOINLINE void* pthread_read_tss(struct pthread_config* config, u32 key) {
    unsigned long tcb = get_tcb_pointer();
    void* pthread = NULL;
    long err = bpf_probe_read_user(&pthread, sizeof(void*), (void*) (tcb + config->struct_pthread_pointer_offset));
    if (err != 0) {
        BPF_TRACE("pthread: failed to read pthread pointer at offset %d: %d", config->struct_pthread_pointer_offset, err);
        return NULL;
    }

    if (pthread == NULL) {
        BPF_TRACE("pthread: pthread pointer is NULL");
        return NULL;
    }

    if (key >= config->keys_max) {
        BPF_TRACE("pthread: key is out of range, key %d, max %d", key, config->keys_max);
        return NULL;
    }

    if (key < config->key_second_level_size) {
        return pthread_read_tss_second_level(&config->key_data, pthread + config->first_specific_block_offset, key);
    }

    void* second_level_block = pthread_read_tss_first_level((void*) (pthread + config->specific_array_offset), key / config->key_second_level_size);
    if (second_level_block == NULL) {
        return NULL;
    }

    return pthread_read_tss_second_level(&config->key_data, second_level_block, key % config->key_second_level_size);
}
