#pragma once

#include "core.h"

#include <bpf/bpf.h>

// A lot of CO-RE magic to find top-level cgroup id.

enum cgroup_limits { PARENT_CGROUP_MAX_LEVELS = 16 };

BTF_EXPORT(enum cgroup_limits);

enum cgroup_consts : u64 {
    END_OF_CGROUP_LIST = -1
};

enum {
    MAX_TRACED_CGROUPS = 16 * 1024,
    MAX_TRACED_PROCESSES = 1024
};

BPF_MAP(traced_cgroups, BPF_MAP_TYPE_HASH, u64, u8, MAX_TRACED_CGROUPS)
BPF_MAP(traced_processes, BPF_MAP_TYPE_HASH, u32, u8, MAX_TRACED_PROCESSES)

static ALWAYS_INLINE struct cgroup* cgroup_parent(struct cgroup* cg) {
    return BPF_CORE_READ(cg, self.parent, cgroup);
}

static ALWAYS_INLINE u64 cgroup_inode(struct cgroup* cg) {
    struct kernfs_node* kn = BPF_CORE_READ(cg, kn);

    if (BPF_CORE_FIELD_EXISTS(kn->id.id)) {
        // 5.4
        return BPF_CORE_READ(kn, id.id);
    } else {
        // 5.15
        struct kernfs_node___v15* new_kn = (void*)kn;
        return BPF_CORE_READ(new_kn, id);
    }
}

static ALWAYS_INLINE void get_current_cgroup_hierarchy_v1(u64 out[PARENT_CGROUP_MAX_LEVELS], u64* parent) {
    struct task_struct* task = (void*)bpf_get_current_task();

    int subsys_id = BPF_CORE_ENUM_VALUE(enum cgroup_subsys_id, freezer_cgrp_id);
    struct css_set* cgroups = BPF_CORE_READ(task, cgroups);
    struct cgroup* cgroup = BPF_CORE_READ(cgroups, subsys[subsys_id], cgroup);

    *parent = END_OF_CGROUP_LIST;
    int i;
    for (i = 0; i < PARENT_CGROUP_MAX_LEVELS;) {
        u64 cgid = cgroup_inode(cgroup);
        void* exists = bpf_map_lookup_elem(&traced_cgroups, &cgid);
        if (exists != 0) {
            *parent = cgid;
            break;
        }
        out[i++] = cgid;

        cgroup = cgroup_parent(cgroup);
        if (cgroup == 0) {
            break;
        }
    }
    if (i < PARENT_CGROUP_MAX_LEVELS) {
        out[i++] = END_OF_CGROUP_LIST;
    }
}

static ALWAYS_INLINE void get_current_cgroup_hierarchy_v2(u64 out[PARENT_CGROUP_MAX_LEVELS], u64* parent) {
    struct task_struct* task = (void*)bpf_get_current_task();

    struct css_set* cgroups = BPF_CORE_READ(task, cgroups);
    struct cgroup* cgroup = BPF_CORE_READ(cgroups, dfl_cgrp);

    *parent = END_OF_CGROUP_LIST;
    int i;
    for (i = 0; i < PARENT_CGROUP_MAX_LEVELS;) {
        u64 cgid = cgroup_inode(cgroup);
        void* exists = bpf_map_lookup_elem(&traced_cgroups, &cgid);
        if (exists != 0) {
            *parent = cgid;
            break;
        }
        out[i++] = cgid;

        cgroup = cgroup_parent(cgroup);
        if (cgroup == 0) {
            break;
        }
    }
    if (i < PARENT_CGROUP_MAX_LEVELS) {
        out[i++] = END_OF_CGROUP_LIST;
    }
}
