#pragma once

#include <bpf/bpf.h>

////////////////////////////////////////////////////////////////////////////////

enum cgroup_subsys_id {
    freezer_cgrp_id,
    CGROUP_SUBSYS_COUNT,
};

struct cgroup_subsys_state {
    struct cgroup* cgroup;
    struct cgroup_subsys_state* parent;
};

struct cgroup {
    struct cgroup_subsys_state self;

    int id;
    int level;
    struct cgroup* parent;
    struct kernfs_node* kn;
    struct cgroup_root* root;
};

union kernfs_node_id {
    struct {
        u32 ino;
        u32 generation;
    };
    u64 id;
};

struct kernfs_node {
    union kernfs_node_id id;
};

struct kernfs_node___v15 {
    u64 id;
};

struct cgroup_root {
    int hierarchy_id;
    char name[64];
};

struct css_set {
    struct cgroup_subsys_state* subsys[CGROUP_SUBSYS_COUNT];
    struct cgroup* dfl_cgrp;
};

////////////////////////////////////////////////////////////////////////////////

enum process_flags {
    PF_KTHREAD = 0x00200000,
};

struct thread_struct {
    unsigned long fsbase;
};

enum {
    TASK_COMM_LEN = 16,
};

////////////////////////////////////////////////////////////////////////////////

struct upid {
    int nr;
    struct pid_namespace* ns;
};

struct pid {
    unsigned int level;
    struct upid numbers[];
};

struct ns_common {
    unsigned int inum;
};

struct pid_namespace {
    struct ns_common ns;
};

////////////////////////////////////////////////////////////////////////////////

struct mm_struct {
    unsigned long start_code;
};

////////////////////////////////////////////////////////////////////////////////

struct task_struct {
    void* stack;
    unsigned int flags;
    struct task_struct* group_leader;
    struct mm_struct* mm;
    u64 real_start_time;
    char comm[TASK_COMM_LEN];
    struct css_set* cgroups;
    struct pid* thread_pid;
    struct thread_struct thread;
};

struct task_struct___v15 {
    u64 start_boottime;
};

////////////////////////////////////////////////////////////////////////////////

struct perf_event_attr___core {
    u32 type;
    u64 config;
    u64 sample_type;
    u64 read_format;
};

struct perf_event {
    struct perf_event_attr___core attr;
    u64 id;
};

struct bpf_perf_event_data_kern {
    struct perf_event* event;
};

////////////////////////////////////////////////////////////////////////////////
