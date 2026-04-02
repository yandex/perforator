#pragma once

#include "metrics.h"
#include "binary.h"
#include "unwind_ctx.h"

// For PERF_MAX_STACK_DEPTH
#include <linux/perf_event.h>

#ifdef __x86_64__

#include "arch/x86/regs.h"
#include "arch/x86/dwarf.h"

#elif __aarch64__

#include "arch/arm/regs.h"
#include "arch/arm/dwarf.h"

#else

#error This arch is not supported by Perforator yet

#endif

////////////////////////////////////////////////////////////////////////////////

enum unwind_table_page_type : u8 {
    UNWIND_TABLE_PAGE_TYPE_EMPTY = 0,
    UNWIND_TABLE_PAGE_TYPE_LEAF = 1,
    UNWIND_TABLE_PAGE_TYPE_NODE = 2,
};

typedef u32 page_id;

enum unwind_page_table_params : u32 {
    UNWIND_TABLE_INVALID_PAGE_ID = (page_id)-1,

    UNWIND_PAGE_TABLE_DEPTH = 3,
    UNWIND_PAGE_TABLE_PAGE_SIZE = 4136,
    UNWIND_PAGE_TABLE_NUM_PAGES_TOTAL = 1024 * 1024 * 1023 / UNWIND_PAGE_TABLE_PAGE_SIZE,
    UNWIND_PAGE_TABLE_NUM_PAGES_PER_PART = (1 << 14),
    UNWIND_PAGE_TABLE_NUM_PARTS = (UNWIND_PAGE_TABLE_NUM_PAGES_TOTAL-1) / UNWIND_PAGE_TABLE_NUM_PAGES_PER_PART + 1,

    UNWIND_PAGE_TABLE_NODE_WIDTH = 10,
    UNWIND_PAGE_TABLE_LEAF_WIDTH = 8,
};

#define POW2(x) (1 << x)

struct unwind_table_page_leaf {
    u32 length;
    u32 pc[POW2(UNWIND_PAGE_TABLE_LEAF_WIDTH)];
    u32 ranges[POW2(UNWIND_PAGE_TABLE_LEAF_WIDTH)];
    struct unwind_rule rules[POW2(UNWIND_PAGE_TABLE_LEAF_WIDTH)];
};

struct unwind_table_page_node {
    page_id children[POW2(UNWIND_PAGE_TABLE_NODE_WIDTH)];
};

struct unwind_table_page {
    page_id id;
    u32 padding;
    u64 begin_address;
    u64 end_address;
    page_id next_page;
    enum unwind_table_page_type type;
    union {
        struct unwind_table_page_leaf leaf;
        struct unwind_table_page_node node;
    } kind;
};

// FIXME: This comment is needed, otherwise compilation fails. BTF_EXPORT uses __LINE__ to distinguish different types
BTF_EXPORT(struct unwind_table_page);

////////////////////////////////////////////////////////////////////////////////
_Static_assert(sizeof(struct unwind_table_page) == UNWIND_PAGE_TABLE_PAGE_SIZE, "");


BPF_MAP_STRUCT(unwind_table_part, BPF_MAP_TYPE_ARRAY, u32, struct unwind_table_page, UNWIND_PAGE_TABLE_NUM_PAGES_PER_PART, 0);


struct {
    BPF_MAP_DEF_UINT(type, BPF_MAP_TYPE_ARRAY_OF_MAPS);
    BPF_MAP_DEF_UINT(key_size, sizeof(u32));
    BPF_MAP_DEF_UINT(max_entries, UNWIND_PAGE_TABLE_NUM_PARTS);
    BPF_MAP_DEF_ARRAY(values, struct unwind_table_part);
} unwind_table SEC(BPF_SEC_BTF_MAPS);

enum {
    MAX_PER_PROCESS_UNWIND_TABLES = 1000,
};

BPF_MAP(binary_unwind_roots, BPF_MAP_TYPE_HASH, binary_id, page_id, MAX_BINARIES);
BPF_MAP(process_unwind_roots, BPF_MAP_TYPE_HASH, u32, page_id, MAX_PER_PROCESS_UNWIND_TABLES);

////////////////////////////////////////////////////////////////////////////////

enum dwarf_unwind_error {
    // No error.
    DWARF_UNWIND_ERROR_NONE = 0,

    // The stack provided in @dwarf_unwind_init was exhausted.
    DWARF_UNWIND_ERROR_TOO_MANY_FRAMES = 1,

    // Failed to locate unwind rule by instruction location.
    // Probably malformed unwind table.
    DWARF_UNWIND_ERROR_NO_RULE_FOR_INSTRUCTION = 2,

    // Failed to evaluate next frame state.
    // Probably unsupported CFI rules.
    // TODO(sskvor): more verbose error codes.
    DWARF_UNWIND_ERROR_RULE_EVALUATION_FAILED = 3,
};

enum { STACK_SIZE = PERF_MAX_STACK_DEPTH };
enum { DWARF_UNWIND_MAX_STACK_SIZE = 128 };

struct stack {
    u32 len;
    u64 ips[STACK_SIZE];
};

struct dwarf_unwind_context {
    u32 pid;
    enum dwarf_unwind_error error;
    struct unwind_context cfi;
    u32 framepointers;
};

BPF_MAP(heap, BPF_MAP_TYPE_PERCPU_ARRAY, u32, struct dwarf_unwind_context, 1);

static ALWAYS_INLINE struct dwarf_unwind_context* dwarf_get_context() {
    u32 zero = 0;
    return bpf_map_lookup_elem(&heap, &zero);
}

////////////////////////////////////////////////////////////////////////////////

static NOINLINE bool locate_rule(struct unwind_table_page_leaf* page, u64 pc, struct unwind_rule* rule) {
    if (page == NULL) {
        return false;
    }

    u32 l = 0;
    u32 r = page->length;
    DWARF_TRACE("start bs: pc=%llx, l=%d, r=%d\n", pc, l, r);

    for (u32 i = 0; i < 8; ++i) {
        u32 m = (r + l) / 2;
        if (m >= 256) {
            return false;
        }
        u64 mpc = page->pc[m];

        if (mpc <= pc) {
            l = m;
        } else {
            r = m;
        }
    }

    if (l >= 256) {
        return false;
    }

    DWARF_TRACE("bs result: %d, from=%llx, to=%llx\n", l, page->pc[l], page->pc[l] + page->ranges[l]);
    if (page->pc[l] > pc || page->pc[l] + page->ranges[l] < pc) {
        return false;
    }

    if (rule != NULL) {
        *rule = page->rules[l];
    }

    return true;
}

static ALWAYS_INLINE struct unwind_table_page* get_unwind_table_page(page_id pageid) {
    u32 part_id = pageid / UNWIND_PAGE_TABLE_NUM_PAGES_PER_PART;
    u32 part_page_id = pageid % UNWIND_PAGE_TABLE_NUM_PAGES_PER_PART;
    struct unwind_table_path* part = bpf_map_lookup_elem(&unwind_table, &part_id);
    if (part == NULL) {
        return NULL;
    }
    return bpf_map_lookup_elem(part, &part_page_id);
}

static NOINLINE struct unwind_table_page* unwind_table_lookup_page(page_id pageid, u64 pc) {
    u64 pc0 = (pc >> 28) & 1023;
    u64 pc1 = (pc >> 18) & 1023;
    u64 pc2 = (pc >> 8) & 1023;

    struct unwind_table_page* page;

#define ADVANCE(stage) \
    page = get_unwind_table_page(pageid); \
    if (page == 0) { \
        BPF_TRACE("unknown stage %d page %d\n", stage, pageid); \
        return false; \
    } \
    if (page->id != pageid) { \
        BPF_TRACE("unexpected page id: %d vs %d\n", page->id, pageid); \
        return false; \
    } \
    if (page->type != UNWIND_TABLE_PAGE_TYPE_NODE) { \
        BPF_TRACE("unexpected page type %d\n", (int)page->type); \
        return false; \
    } \
    DWARF_TRACE("page children: [%d,%d,%d,...]\n", page->kind.node.children[0], page->kind.node.children[1], page->kind.node.children[2]); \
    DWARF_TRACE("page children: [%d,%d,%d,...]\n", page->kind.node.children[3], page->kind.node.children[4], page->kind.node.children[5]); \
    DWARF_TRACE("lookup page %d[%d] -> %d\n", pageid, CAT(pc, stage), page->kind.node.children[CAT(pc, stage)]); \
    pageid = page->kind.node.children[CAT(pc, stage)];

    ADVANCE(0)
    ADVANCE(1)
    ADVANCE(2)

#undef ADVANCE

    page = get_unwind_table_page(pageid);
    if (page == 0 || page->type != UNWIND_TABLE_PAGE_TYPE_LEAF) {
        BPF_TRACE("unknown leaf page %d\n", pageid);
        return NULL;
    }

    BPF_TRACE("found page %d [%llx, %llx)\n", pageid, page->begin_address, page->end_address);
    if (page->end_address <= pc) {
        BPF_TRACE("trying next page %d\n", page->next_page);
        page = get_unwind_table_page(page->next_page);
        if (page == 0 || page->type != UNWIND_TABLE_PAGE_TYPE_LEAF) {
            BPF_TRACE("unknown leaf page %d\n", pageid);
            return NULL;
        }
    }

    struct unwind_table_page_leaf* leaf = &page->kind.leaf;
    BPF_TRACE("found leaf %d [%llx, %llx)\n", pageid, leaf->pc[0], leaf->pc[255]);
#ifndef BPF_DEBUG
    (void) leaf;
#endif
    return page;
}

static NOINLINE bool unwind_table_lookup_fast(page_id pageid, u64 pc, struct unwind_rule* rule) {
    struct unwind_table_page* leaf_page = unwind_table_lookup_page(pageid, pc);
    if (leaf_page == NULL) {
        return false;
    }
    BPF_TRACE("adjusting pc by page begin_address for binary search: %llx - %llx = %llx\n", pc, leaf_page->begin_address, pc-leaf_page->begin_address);
    return locate_rule(&leaf_page->kind.leaf, pc-leaf_page->begin_address, rule);
}

////////////////////////////////////////////////////////////////////////////////

BTF_EXPORT(enum unwind_page_table_params);

////////////////////////////////////////////////////////////////////////////////

static ALWAYS_INLINE bool dwarf_cfi_eval_cfa(
    struct unwind_context* prev,
    struct unwind_context* next,
    struct cfa_unwind_rule* rule
) {
    if (rule == NULL || prev == NULL || next == NULL) {
        return false;
    }

    switch (rule->kind) {
    case UNWIND_RULE_REGISTER_OFFSET: {
        switch (rule->regno) {
        case DWARF_CFI_STACK_REGISTER_NUMBER:
            DWARF_TRACE("Found rule UNWIND_RULE_REGISTER_OFFSET register sp+%d\n", (int)rule->offset);
            if (prev->cfa == DWARF_CFI_UNKNOWN_REGISTER) {
                DWARF_TRACE("Failed to eval CFA: SP is unknown\n");
                return false;
            }
            next->cfa = prev->cfa + rule->offset;
            DWARF_TRACE("Set sp to %llx=%llx+%llx\n", next->cfa, prev->cfa, rule->offset);
            return true;
        case DWARF_CFI_FRAME_REGISTER_NUMBER:
            DWARF_TRACE("Found rule UNWIND_RULE_REGISTER_OFFSET register fp+%d\n", (int)rule->offset);
            if (prev->fp == DWARF_CFI_UNKNOWN_REGISTER) {
                DWARF_TRACE("Failed to eval CFA: FP is unknown\n");
                return false;
            }
            next->cfa = prev->fp + rule->offset;
            DWARF_TRACE("Set cf to %llx=%llx+%llx\n", next->cfa, prev->fp, rule->offset);
            return true;

        default:
            DWARF_TRACE("Unsupported cfa rule register %d\n", (int)rule->regno);
            return false;
        }
    }
    default:
        DWARF_TRACE("Unsupported cfa rule kind %d\n", (int)rule->kind);
        return false;
    }
}

ALWAYS_INLINE bool dwarf_cfi_eval_fp(
    struct unwind_context* prev,
    struct unwind_context* next,
    struct rbp_unwind_rule* rule
) {
    if (rule->offset == DWARF_UNWIND_CFA_RULE_UNDEFINED) {
        DWARF_TRACE("Undefined FP rule, using prev FP\n");
        next->fp = prev->fp;
        return true;
    }

    u64 address = next->cfa + rule->offset;
    DWARF_TRACE("Found fp offset %d, location %llx\n", rule->offset, address);

    int err = bpf_probe_read_user(&next->fp, sizeof(next->fp), (void*)address);
    if (err != 0) {
        DWARF_TRACE("bpf_probe_read_user failed: %d\n", err);
        return false;
    }

    return true;
}

static NOINLINE bool dwarf_cfi_eval(
    struct unwind_context* prev,
    struct unwind_context* next,
    struct unwind_rule* rule
) {
    if (!dwarf_cfi_eval_cfa(prev, next, &rule->cfa)) {
        DWARF_TRACE("failed to eval cfa\n");
        return false;
    }
    if (!dwarf_cfi_eval_fp(prev, next, &rule->rbp)) {
        DWARF_TRACE("failed to eval rbp\n");
        return false;
    }
    if (!dwarf_cfi_eval_ra(prev, next, &rule->ra)) {
        DWARF_TRACE("failed to eval ra\n");
        return false;
    }
    return true;
}

// Initialize @ctx.
static NOINLINE bool dwarf_unwind_init(
    struct dwarf_unwind_context* ctx,
    struct user_regs* regs,
    u32 pid
) {
    ctx->pid = pid;
    ctx->error = DWARF_UNWIND_ERROR_NONE;
    ctx->framepointers = 0;

    dwarf_unwind_setup_userspace_registers(&ctx->cfi, regs);
    BPF_TRACE("Initialize dwarf ip to %llx\n", ctx->cfi.ip);
    return true;
}

////////////////////////////////////////////////////////////////////////////////

struct executable_mapping_trie_key {
    u32 prefixlen;
    u32 pid;
    u64 address_prefix;
};

struct executable_mapping_info {
    u32 id;
};

struct executable_mapping_key {
    u32 pid;
    u32 unused_padding;
    u32 id;
};

struct executable_mapping {
    u64 begin;
    u64 end;
    u64 binary_id;
    i64 offset;
};

enum executable_mapping_table_params : u64 {
    EXECUTABLE_MAPPING_LPM_TRIE_SIZE = 1024 * 1024,
    EXECUTABLE_MAPPING_TABLE_SIZE = 256 * 1024,
};

BPF_MAP_F(
    executable_mapping_trie,
    BPF_MAP_TYPE_LPM_TRIE,
    struct executable_mapping_trie_key,
    struct executable_mapping_info,
    EXECUTABLE_MAPPING_LPM_TRIE_SIZE,
    BPF_F_NO_PREALLOC
);

BPF_MAP(
    executable_mappings,
    BPF_MAP_TYPE_HASH,
    struct executable_mapping_key,
    struct executable_mapping,
    EXECUTABLE_MAPPING_TABLE_SIZE
);

static NOINLINE struct executable_mapping* dwarf_unwind_locate_executable(u32 pid, u64 ip) {
    struct executable_mapping_trie_key trie_key = {
        .prefixlen = 96,
        .pid = pid,
        .address_prefix = __bpf_cpu_to_be64(ip),
    };

    struct executable_mapping_info* info = bpf_map_lookup_elem(&executable_mapping_trie, &trie_key);
    if (!info) {
        metric_increment(METRIC_DWARF_ERROR_MAPPING_LPMTRIE_MISS_COUNT);
        return NULL;
    }

    struct executable_mapping_key key = {
        .pid = pid,
        .unused_padding = 0,
        .id = info->id,
    };

    struct executable_mapping* mapping = bpf_map_lookup_elem(&executable_mappings, &key);
    if (!mapping) {
        metric_increment(METRIC_DWARF_ERROR_MAPPING_LPMTRIE_NOMAPPING_COUNT);
        return NULL;
    }

    if (mapping->begin > ip || mapping->end <= ip) {
        metric_increment(METRIC_DWARF_ERROR_MAPPING_LPMTRIE_MALFORMED_COUNT);
        BPF_TRACE("Malformed mapping found for rip %llx: [%llx, %llx)\n", ip, mapping->begin, mapping->end);
        return NULL;
    }

    return mapping;
}

////////////////////////////////////////////////////////////////////////////////

ALWAYS_INLINE u32 dwarf_unwind_get_executable_root(binary_id bid) {
    u32* page = bpf_map_lookup_elem(&binary_unwind_roots, &bid);
    if (page == NULL) {
        DWARF_TRACE("failed to lookup mapping %llu root\n", bid);
        return -1;
    }

    DWARF_TRACE("found mapping %llu root %d\n", bid, *page);
    return *page;
}

ALWAYS_INLINE u32 dwarf_unwind_get_process_root(u32 pid) {
    u32* page = bpf_map_lookup_elem(&process_unwind_roots, &pid);
    if (page == NULL) {
        DWARF_TRACE("failed to lookup process %d root\n", pid);
        return -1;
    }

    DWARF_TRACE("found process %d root %d\n", pid, *page);
    return *page;
}

static NOINLINE bool dwarf_unwind_locate_rule(
    struct dwarf_unwind_context* ctx,
    struct unwind_rule* rule
) {
    if (ctx == NULL || rule == NULL) {
        return false;
    }

    u64 rip = ctx->cfi.ip;
    DWARF_TRACE("start locate rule for rip %llx\n", rip);

    struct executable_mapping* mapping = dwarf_unwind_locate_executable(ctx->pid, rip);
    if (!mapping) {
        u32 process_root = dwarf_unwind_get_process_root(ctx->pid);
        if (process_root != (u32)-1) {
            bool ok = unwind_table_lookup_fast(process_root, rip, rule);
            if (ok) {
                DWARF_TRACE("found per-process unwind rule for rip %llx\n", rip);
                return true;
            }
            DWARF_TRACE("per-process unwind table also lacks rip %llx\n", rip);
            metric_increment(METRIC_DWARF_ERROR_PER_PROCESS_RULE_LOOKUP_COUNT);
        }
        metric_increment(METRIC_DWARF_ERROR_MAPPING_LOCATE_COUNT);
        DWARF_TRACE("no mapping found for rip %llx\n", rip);
        return false;
    }

    binary_id bid = mapping->binary_id;
    if (bid == -1) {
        metric_increment(METRIC_DWARF_ERROR_MAPPING_NOBINARYID_COUNT);
        DWARF_TRACE("mapping binary id is not set\n");
        return false;
    }
    DWARF_TRACE("found binary %llu for rip %llx, elf adjusted pc: %llx\n", mapping->binary_id, rip, rip - mapping->offset);

    u32 root = dwarf_unwind_get_executable_root(mapping->binary_id);
    if (root == (u32)-1) {
        metric_increment(METRIC_DWRAF_ERROR_MAPPING_NOBINARYROOT_COUNT);
        DWARF_TRACE("no mapping found for root %llx\n", root);
        return false;
    }

    if (!unwind_table_lookup_fast(root, rip - mapping->offset, rule)) {
        metric_increment(METRIC_DWARF_ERROR_MAPPING_UNWINDTABLELOOKUP_COUNT);
        DWARF_TRACE("unwind table lookup failed\n");
        return false;
    }

    return true;
}

static bool NOINLINE dwarf_unwind_record_stack_frame(
    struct dwarf_unwind_context* ctx,
    struct stack* stack
) {
    if (stack == NULL || ctx == NULL) {
        return false;
    }

    if (stack->len >= DWARF_UNWIND_MAX_STACK_SIZE || stack->len >= STACK_SIZE) {
        metric_increment(METRIC_DWARF_ERROR_TOOMANYFRAMES_COUNT);
        ctx->error = DWARF_UNWIND_ERROR_TOO_MANY_FRAMES;
        return false;
    } else {
        stack->ips[stack->len++] = ctx->cfi.ip;
    }

    return true;
}

// Perform one step of dwarf unwinding.
// This function contains heavy loop and is called inside heavy loop, so it MUST be marked with NOINLINE.
// The verifier struggles with the outermost loop otherwise.
enum dwarf_unwind_step_result NOINLINE dwarf_unwind_step() {
    struct dwarf_unwind_context* ctx = dwarf_get_context();
    if (ctx == NULL) {
        DWARF_TRACE("failed to lookup dwarf_unwind_context\n");
        return DWARF_UNWIND_STEP_FAILED;
    }

    /*
    // According to System V AMD64 ABI (3.4.1 Initial Stack and Register State),
    // User-space runtime should mark the deepest frame with %rbp set to zero.
    // https://refspecs.linuxbase.org/elf/x86_64-abi-0.99.pdf
    if (ctx->cfi.rbp == 0) {
        DWARF_TRACE("reached bottom of the stack\n");
        return DWARF_UNWIND_STEP_FINISHED;
    }
    */

    // Locate unwind table rule for that %rip using page table search.
    struct unwind_rule rule = {};
    if (!dwarf_unwind_locate_rule(ctx, &rule)) {
        // Try to unwind one frame using frame pointers.
        metric_increment(METRIC_DWARF_ERROR_NORULEFORINSTRUCTION_COUNT);
        DWARF_TRACE("failed to locate rule, try fp\n");
        return dwarf_unwind_step_fp(&ctx->cfi, &ctx->framepointers);
    }

#ifdef TRACE_DWARF_UNWINDING
    u64 serialized = 0;
    memcpy(&serialized, &rule, 7);
    DWARF_TRACE("found dwarf rule %llx\n", serialized);
#endif

    // Evaluate next frame.
    struct unwind_context next;
    dwarf_cfi_context_init_next(&next);

    if (!dwarf_cfi_eval(&ctx->cfi, &next, &rule)) {
        DWARF_TRACE("failed to evaluate CFI rule\n");
        metric_increment(METRIC_DWARF_ERROR_RULEEVALUATIONFAILED_COUNT);
        ctx->error = DWARF_UNWIND_ERROR_RULE_EVALUATION_FAILED;
        return DWARF_UNWIND_STEP_FAILED;
    }
    DWARF_TRACE("next regs state: cfa=%lx\n", next.cfa);
    DWARF_TRACE("next regs state: fp=%lx\n", next.fp);
    DWARF_TRACE("next regs state: ip=%lx\n", next.ip);

    ctx->cfi = next;

    // Done!
    if (ctx->cfi.ip == (u64)-1) {
        return DWARF_UNWIND_STEP_FINISHED;
    }

    return DWARF_UNWIND_STEP_CONTINUE;
}
