#pragma once

#include "contract/wire_types.inc"
#include "cgroups.h"
#include "thread_local.h"
#include "lbr.h"
#include "interpreter/types.h"
#include "thread_local.h"
#include "jvm/api.h"

#include <linux/perf_event.h>
#include <bpf/bpf.h>

////////////////////////////////////////////////////////////////////////////////

BPF_MAP(samples, BPF_MAP_TYPE_PERF_EVENT_ARRAY, u32, u32, 0);
BPF_MAP(processes, BPF_MAP_TYPE_PERF_EVENT_ARRAY, u32, u32, 0);

////////////////////////////////////////////////////////////////////////////////

// Export types used by the Go parser (btf2go generates Go structs + GetSection
// accessor + compile-time offset assertions from these).
BTF_EXPORT(struct jvm_stack);
BTF_EXPORT(struct tls_collect_result);
BTF_EXPORT(struct last_branch_records);
BTF_EXPORT(struct section_desc);
BTF_EXPORT(struct record_sample_header);
BTF_EXPORT(struct language_section_header);
BTF_EXPORT(enum language_id);
BTF_EXPORT(enum language_payload_kind);

struct packed_sample {
    struct record_sample_header header;
    u8 data[PACKED_SAMPLE_MAX_DATA];
};

////////////////////////////////////////////////////////////////////////////////

#define BPF_PERFBUF_SUBMIT(map, var) \
    long res = bpf_perf_event_output(ctx, &map, BPF_F_CURRENT_CPU, var, sizeof(*var)); \
    if (res != 0) { \
        BPF_TRACE("bpf_perf_event_output failed: %ld\n", res); \
    } \

void submit_packed_sample(void* ctx, struct packed_sample* packed, u32 size) {
    packed->header.tag = RECORD_TAG_SAMPLE;
    if (size > sizeof(struct packed_sample)) {
        size = sizeof(struct packed_sample);
    }
#ifdef PERFORATOR_COMPAT_5_4
    long res = bpf_perf_event_output(ctx, &samples, BPF_F_CURRENT_CPU, packed, sizeof(struct packed_sample));
#else
    long res = bpf_perf_event_output(ctx, &samples, BPF_F_CURRENT_CPU, packed, size);
#endif
    if (res != 0) {
        BPF_TRACE("bpf_perf_event_output failed: %ld\n", res);
    }
}

void submit_new_process(void* ctx, struct record_new_process* rec) {
    rec->tag = RECORD_TAG_NEW_PROCESS;
    BPF_PERFBUF_SUBMIT(processes, rec);
}

////////////////////////////////////////////////////////////////////////////////
