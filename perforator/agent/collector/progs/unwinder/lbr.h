#pragma once

#include <bpf/bpf.h>

enum {
    // At the time of writing Intel processors support up to 32 LBR entries
    MAX_BRANCH_RECORDS = 32
};

// Mimics struct perf_branch_entry
struct branch_record {
    u64 from;
    u64 to;
    u64 flags;
};

struct last_branch_records {
    u64 nr;
    struct branch_record entries[MAX_BRANCH_RECORDS];
};

static ALWAYS_INLINE void collect_lbr_stack(void* ctx, struct last_branch_records* records) {
    // Available since 5.7 kernel
    if (bpf_core_enum_value_exists(enum bpf_func_id, BPF_FUNC_read_branch_records)) {
        int written = bpf_read_branch_records(ctx, records->entries, sizeof(records->entries), 0);
        // We are fine with reading 0 entries
        if (written < 0) {
            BPF_TRACE("failed to read last branch records, errno: %d", written);
            records->nr = 0;
            return;
        }

        records->nr = written / sizeof(struct branch_record);
        BPF_TRACE("successfully read %d last branch records", records->nr);
    } else {
        records->nr = 0;
    }
}
