#pragma once

#include <bpf/bpf.h>

#include <linux/bpf.h>
#include <linux/perf_event.h>

#include "output.h"

enum {
    MAX_ADDITIONAL_EVENTS = 5,
};

BPF_MAP_STRUCT(additional_event_map, BPF_MAP_TYPE_PERF_EVENT_ARRAY, u32, u32, 0, 0);

struct {
    BPF_MAP_DEF_UINT(type, BPF_MAP_TYPE_ARRAY_OF_MAPS);
    BPF_MAP_DEF_UINT(key_size, sizeof(u32));
    BPF_MAP_DEF_UINT(max_entries, MAX_ADDITIONAL_EVENTS);
    BPF_MAP_DEF_ARRAY(values, struct additional_event_map);
} additional_events SEC(BPF_SEC_BTF_MAPS);

// Per-CPU baselines of the additional events, updated on every read.
// prev[i].running == 0 means "no baseline yet".
struct additional_event_state {
    struct bpf_perf_event_value prev[MAX_ADDITIONAL_EVENTS];
};

struct {
    BPF_MAP_DEF_UINT(type, BPF_MAP_TYPE_PERCPU_ARRAY);
    BPF_MAP_DEF_UINT(key_size, sizeof(u32));
    BPF_MAP_DEF_UINT(value_size, sizeof(struct additional_event_state));
    BPF_MAP_DEF_UINT(max_entries, 1);
} additional_event_states SEC(BPF_SEC_BTF_MAPS);

static NOINLINE void profiler_stage_collect_additional_events(struct record_sample_header* sample) {
#ifndef PERFORATOR_COMPAT_5_4
    u32 key = 0;
    struct additional_event_state* state = bpf_map_lookup_elem(&additional_event_states, &key);
    if (state == NULL) {
        return;
    }

#pragma unroll
    for (int i = 0; i < MAX_ADDITIONAL_EVENTS; i++) {
        struct bpf_perf_event_value* prev = &state->prev[i];
        struct perf_event_value* out = &sample->additional_events[i];

        struct perf_event_value delta = {};
        key = i;
        void* map = bpf_map_lookup_elem(&additional_events, &key);
        if (map != NULL) {
            struct bpf_perf_event_value value = {};
            long err = bpf_perf_event_read_value(map, BPF_F_CURRENT_CPU, &value, sizeof(value));
            if (err == 0 && value.running > 0) {
                if (prev->running == 0) {
                    BPF_TRACE("additional_events[%d]: first read, baseline set\n", i);
                    *prev = value;
                } else if (value.counter < prev->counter || value.enabled < prev->enabled || value.running < prev->running) {
                    BPF_TRACE("additional_events[%d]: counters went backwards, re-baselining\n", i);
                    *prev = value;
                } else {
                    delta.counter = value.counter - prev->counter;
                    delta.enabled = value.enabled - prev->enabled;
                    delta.running = value.running - prev->running;
                    *prev = value;
                    if (delta.running == 0) {
                        BPF_TRACE("additional_events[%d]: event did not run during the window\n", i);
                        delta.counter = 0;
                        delta.enabled = 0;
                    }
                }
            } else if (err != -2) {
                // -2 (-ENOENT) is the expected steady state for CPUs without
                // event configured.
                BPF_TRACE("bpf_perf_event_read_value failed: %d\n", err);
            }
        }
        *out = delta;
    }
    long err = bpf_map_update_elem(&additional_event_states, &key, state, BPF_ANY);
    if (err != 0) {
        BPF_TRACE("bpf_map_update_elem failed: %d\n", err);
    }
#endif
}
