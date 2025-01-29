#include <bpf/bpf.h>

struct mapping_trie_key {
    u32 prefixlen;
    u32 pid;
    u64 address_prefix;
};

struct mapping_info {
    u64 begin;
    u64 end;
    u64 bias;
    u64 binary_id;
};

BPF_MAP_F(trie, BPF_MAP_TYPE_LPM_TRIE, struct mapping_trie_key, struct mapping_info, 1024 * 1024, BPF_F_NO_PREALLOC);

SEC("tracepoint/sched/sched_switch")
int trace_sched_switch(void* ctx) {
    u32 pid = bpf_get_current_pid_tgid() >> 32;

    struct mapping_trie_key key;
    key.prefixlen = 96;
    key.pid = pid;
    key.address_prefix = __bpf_cpu_to_be64(0xdeadbeefdeadbeefull);

    struct mapping_info* mapping = bpf_map_lookup_elem(&trie, &key);
    if (!mapping) {
        BPF_TRACE("Failed to find mapping\n");
    } else {
        BPF_TRACE("Found mapping: id=%lld, [%llx, %llx)\n", mapping->binary_id, mapping->begin, mapping->end);
    }

    return 0;
}

LICENSE("GPL")
