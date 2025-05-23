#include <bpf/bpf.h>

struct cgroup {
    int id;
};

struct trace_cgroup_args {
    unsigned long cgrp;
    unsigned long path;
};

SEC("raw_tracepoint/cgroup_mkdir")
int trace_cgroup_mkdir(struct trace_cgroup_args* ctx) {
    struct cgroup* cgrp = (struct cgroup*)ctx->cgrp;
    if (!cgrp) {
        return 0;
    }

    int id = BPF_CORE_READ(cgrp, id);
    BPF_PRINTK("cgroup_mkdir, id: %d, path: %s\n", id, (const char*)ctx->path);

    return 0;
}

LICENSE("GPL")
