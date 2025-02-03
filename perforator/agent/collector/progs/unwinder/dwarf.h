#pragma once

#include "metrics.h"
#include "binary.h"

#include <bpf/bpf.h>

// For PERF_MAX_STACK_DEPTH
#include <linux/perf_event.h>

#include "arch/x86/regs.h"


#define TRACE_DWARF_UNWINDING
#ifdef TRACE_DWARF_UNWINDING
#define DWARF_TRACE(...) BPF_TRACE(__VA_ARGS__)
#else
#define DWARF_TRACE(...)
#endif

enum unwind_rule_kind : u8 {
    UNWIND_RULE_UNSUPPORTED = 0,
    UNWIND_RULE_CFA_MINUS_8 = 1,
    UNWIND_RULE_CFA_PLUS_OFFSET = 2,
    UNWIND_RULE_REGISTER_OFFSET = 3,
    UNWIND_RULE_REGISTER_DERREF_OFFSET = 4,
    UNWIND_RULE_PLT_SECTION = 5,
    UNWIND_RULE_CONSTANT = 6,
};

struct  __attribute__((packed)) cfa_unwind_rule {
    enum unwind_rule_kind kind;
    u8 regno;
    i32 offset;
};

enum {
    DWARF_UNWIND_RBP_RULE_UNDEFINED = 0x7f,
};

struct rbp_unwind_rule {
    // Offset from the CFA to read saved value of RBP from.
    // Now we support only one unwind rule for rbp: deref(CFA+offset).
    i8 offset;
};

struct __attribute__((packed)) unwind_rule {
    struct cfa_unwind_rule cfa;
    struct rbp_unwind_rule rbp;
};

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
    UNWIND_PAGE_TABLE_PAGE_SIZE = 4128,
    UNWIND_PAGE_TABLE_NUM_PAGES_TOTAL = 1024 * 1024 * 1023 / UNWIND_PAGE_TABLE_PAGE_SIZE,
    UNWIND_PAGE_TABLE_NUM_PAGES_PER_PART = (1 << 14),
    UNWIND_PAGE_TABLE_NUM_PARTS = (UNWIND_PAGE_TABLE_NUM_PAGES_TOTAL-1) / UNWIND_PAGE_TABLE_NUM_PAGES_PER_PART + 1,

    UNWIND_PAGE_TABLE_LEVEL_0_WIDTH = 10,
    UNWIND_PAGE_TABLE_LEVEL_1_WIDTH = 10,
    UNWIND_PAGE_TABLE_LEVEL_2_WIDTH = 9,
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
    page_id children[POW2(UNWIND_PAGE_TABLE_LEVEL_0_WIDTH)];
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

BPF_MAP(unwind_roots, BPF_MAP_TYPE_HASH, binary_id, page_id, MAX_BINARIES);

////////////////////////////////////////////////////////////////////////////////

NOINLINE bool locate_rule(struct unwind_table_page_leaf* page, u64 pc, struct unwind_rule* rule) {
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
    if (page->pc[l] + page->ranges[l] < pc) {
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

static NOINLINE struct unwind_table_page_leaf* unwind_table_lookup_page(page_id pageid, u64 pc) {
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
    return leaf;
}

static NOINLINE bool unwind_table_lookup_fast(page_id pageid, u64 pc, struct unwind_rule* rule) {
    struct unwind_table_page_leaf* leaf = unwind_table_lookup_page(pageid, pc);
    if (leaf == NULL) {
        return false;
    }
    return locate_rule(leaf, pc, rule);
}

////////////////////////////////////////////////////////////////////////////////

BTF_EXPORT(enum unwind_page_table_params);

////////////////////////////////////////////////////////////////////////////////

enum { DWARF_CFI_UNKNOWN_REGISTER = 0xfffffffffffffffd };

struct dwarf_cfi_context {
    u64 rsp;
    u64 rbp;
    u64 rip;
};

static ALWAYS_INLINE bool dwarf_cfi_eval_cfa(
    struct dwarf_cfi_context* prev,
    struct dwarf_cfi_context* next,
    struct cfa_unwind_rule* rule
) {
    if (rule == NULL || prev == NULL || next == NULL) {
        return false;
    }

    switch (rule->kind) {
    case UNWIND_RULE_REGISTER_OFFSET: {
        switch (rule->regno) {
        case 7:
            DWARF_TRACE("Found rule UNWIND_RULE_REGISTER_OFFSET register rsp+%d\n", (int)rule->offset);
            if (prev->rsp == DWARF_CFI_UNKNOWN_REGISTER) {
                DWARF_TRACE("Failed to eval CFA: RSP is unknown\n");
                return false;
            }
            next->rsp = prev->rsp + rule->offset;
            DWARF_TRACE("Set rsp to %llx=%llx+%llx\n", next->rsp, prev->rsp, rule->offset);
            return true;

        case 6:
            DWARF_TRACE("Found rule UNWIND_RULE_REGISTER_OFFSET register rbp+%d\n", (int)rule->offset);
            if (prev->rbp == DWARF_CFI_UNKNOWN_REGISTER) {
                DWARF_TRACE("Failed to eval CFA: RBP is unknown\n");
                return false;
            }
            next->rsp = prev->rbp + rule->offset;
            DWARF_TRACE("Set rsp to %llx=%llx+%llx\n", next->rsp, prev->rbp, rule->offset);
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

ALWAYS_INLINE bool dwarf_cfi_eval_rbp(
    struct dwarf_cfi_context* prev,
    struct dwarf_cfi_context* next,
    struct rbp_unwind_rule* rule
) {
    if (rule->offset == DWARF_UNWIND_RBP_RULE_UNDEFINED) {
        DWARF_TRACE("Undefined RBP rule, using previous value\n");
        next->rbp = prev->rbp;
        return true;
    }

    u64 address = next->rsp + rule->offset;
    DWARF_TRACE("Found rbp offset %d, location %llx\n", rule->offset, address);

    int err = bpf_probe_read_user(&next->rbp, sizeof(next->rbp), (void*)address);
    if (err != 0) {
        DWARF_TRACE("bpf_probe_read_user failed: %d\n", err);
        return false;
    }

    return true;
}

ALWAYS_INLINE bool read_return_address(void* location, u64* ra) {
    DWARF_TRACE("read_return_address: read RA from %p\n", location);
    int err = bpf_probe_read_user(ra, sizeof(*ra), location);
    if (err != 0) {
        DWARF_TRACE("read_return_address: bpf_probe_read_user failed: %d\n", err);
        return false;
    }

    // Return address points to the next instruction after the call, not the call itself.
    // So we need to adjust return address to point to the real call instruction.
    *ra -= 1;

    return true;
}

ALWAYS_INLINE bool dwarf_cfi_eval_ra(
    struct dwarf_cfi_context* prev,
    struct dwarf_cfi_context* next
) {
    u64 address = next->rsp - 8;
    return read_return_address((void*)address, &next->rip);
}

NOINLINE bool dwarf_cfi_eval(
    struct dwarf_cfi_context* prev,
    struct dwarf_cfi_context* next,
    struct unwind_rule* rule
) {
    if (!dwarf_cfi_eval_cfa(prev, next, &rule->cfa)) {
        DWARF_TRACE("failed to eval cfa\n");
        return false;
    }
    if (!dwarf_cfi_eval_rbp(prev, next, &rule->rbp)) {
        DWARF_TRACE("failed to eval rbp\n");
        return false;
    }
    if (!dwarf_cfi_eval_ra(prev, next)) {
        DWARF_TRACE("failed to eval ra\n");
        return false;
    }
    return true;
}

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
    struct dwarf_cfi_context cfi;
    u32 framepointers;
};

////////////////////////////////////////////////////////////////////////////////

static ALWAYS_INLINE void dwarf_unwind_setup_userspace_registers(
    struct dwarf_unwind_context* ctx,
    struct user_regs* regs
) {
    ctx->cfi.rsp = regs->rsp;
    ctx->cfi.rbp = regs->rbp;
    ctx->cfi.rip = regs->rip;
}

// Initialize @ctx.
static NOINLINE bool dwarf_unwind_init(
    struct dwarf_unwind_context* ctx,
    struct user_regs* regs
) {
    ctx->pid = bpf_get_current_pid_tgid() >> 32;
    ctx->error = DWARF_UNWIND_ERROR_NONE;
    ctx->framepointers = 0;

    dwarf_unwind_setup_userspace_registers(ctx, regs);
    BPF_TRACE("Initialize dwarf rip to %llx\n", ctx->cfi.rip);
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
    u32* page = bpf_map_lookup_elem(&unwind_roots, &bid);
    if (page == 0) {
        DWARF_TRACE("failed to lookup mapping %llu root\n", bid);
        return -1;
    }

    DWARF_TRACE("found mapping %llu root %d\n", bid, *page);
    return *page;
}

static NOINLINE bool dwarf_unwind_locate_rule(
    struct dwarf_unwind_context* ctx,
    struct unwind_rule* rule
) {
    if (ctx == NULL || rule == NULL) {
        return false;
    }

    u64 rip = ctx->cfi.rip;
    DWARF_TRACE("start locate rule for rip %llx\n", rip);

    struct executable_mapping* mapping = dwarf_unwind_locate_executable(ctx->pid, rip);
    if (!mapping) {
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

enum dwarf_unwind_step_result {
    DWARF_UNWIND_STEP_CONTINUE = 0,
    DWARF_UNWIND_STEP_FINISHED = 1,
    DWARF_UNWIND_STEP_FAILED = 2,
};

// Canonical function prologue and epilogue:
// foo:
//      push %rbp
//      mov %rsp, %rbp
//      ...
//      mov %rbp, %rsp
//      pop %rbp
//      ret
//
// Stack layout:
// rsp0    -> [....]
// rsp0-8  -> [ra0 ]
// rsp0-16 -> [rbp0]
// ...
// rsp1    -> [....]
// rsp1-8  -> [ra1 ]
// rsp1-16 -> [rbp1]
// ...
// rsp2    -> [....]
// rsp2-8  -> [ra2 ]
// rsp2-16 -> [rbp2]
enum dwarf_unwind_step_result NOINLINE dwarf_unwind_step_fp(
    struct dwarf_unwind_context* ctx
) {
    if (ctx == NULL) {
        return DWARF_UNWIND_STEP_FAILED;
    }

    ctx->framepointers++;

    if (!read_return_address((void*)(ctx->cfi.rbp + 8), &ctx->cfi.rip)) {
        metric_increment(METRIC_FP_ERROR_READ_RETURNADDRESS_COUNT);
        ctx->error = DWARF_UNWIND_ERROR_RULE_EVALUATION_FAILED;
        return DWARF_UNWIND_STEP_FAILED;
    }

    // We need to restore rsp in order to support mixed dwarf and frame pointers unwinding.
    // Previous rsp is equal to current rbp + 2*sizeof(register): look at the stack layout.
    ctx->cfi.rsp = ctx->cfi.rbp + 16;

    u64 prev_rbp = 0;
    int err = bpf_probe_read_user(&prev_rbp, sizeof(prev_rbp), (void*)ctx->cfi.rbp);
    if (err != 0) {
        DWARF_TRACE("fp: bpf_probe_read_user failed: %d\n", err);
        metric_increment(METRIC_FP_ERROR_READ_BASEPOINTER_COUNT);
        ctx->error = DWARF_UNWIND_ERROR_RULE_EVALUATION_FAILED;
        return DWARF_UNWIND_STEP_FAILED;
    }
    ctx->cfi.rbp = prev_rbp;

    return DWARF_UNWIND_STEP_CONTINUE;
}

// Perform one step of dwarf unwinding.
// This function contains heavy loop and is called inside heavy loop, so it MUST be marked with NOINLINE.
// The verifier struggles with the outermost loop otherwise.
static enum dwarf_unwind_step_result NOINLINE dwarf_unwind_step(
    struct dwarf_unwind_context* ctx,
    struct stack* stack
) {
    if (stack == NULL || ctx == NULL) {
        return DWARF_UNWIND_STEP_FAILED;
    }

    if (stack->len >= DWARF_UNWIND_MAX_STACK_SIZE || stack->len >= STACK_SIZE) {
        metric_increment(METRIC_DWARF_ERROR_TOOMANYFRAMES_COUNT);
        ctx->error = DWARF_UNWIND_ERROR_TOO_MANY_FRAMES;
        return DWARF_UNWIND_STEP_FAILED;
    } else {
        stack->ips[stack->len++] = ctx->cfi.rip;
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
        return dwarf_unwind_step_fp(ctx);
    }

#ifdef TRACE_DWARF_UNWINDING
    u64 serialized = 0;
    memcpy(&serialized, &rule, 7);
    DWARF_TRACE("found dwarf rule %llx\n", serialized);
#endif

    // Evaluate next frame.
    struct dwarf_cfi_context next = {
        .rsp = DWARF_CFI_UNKNOWN_REGISTER,
        .rbp = DWARF_CFI_UNKNOWN_REGISTER,
        .rip = DWARF_CFI_UNKNOWN_REGISTER,
    };
    if (!dwarf_cfi_eval(&ctx->cfi, &next, &rule)) {
        DWARF_TRACE("failed to evaluate CFI rule\n");
        metric_increment(METRIC_DWARF_ERROR_RULEEVALUATIONFAILED_COUNT);
        ctx->error = DWARF_UNWIND_ERROR_RULE_EVALUATION_FAILED;
        return DWARF_UNWIND_STEP_FAILED;
    }
    DWARF_TRACE("next regs state: ip=%lx\n", next.rip);
    DWARF_TRACE("next regs state: sp=%lx\n", next.rsp);
    DWARF_TRACE("next regs state: bp=%lx\n", next.rbp);
    ctx->cfi = next;

    // Done!
    if (ctx->cfi.rip == (u64)-1) {
        return DWARF_UNWIND_STEP_FINISHED;
    }

    return DWARF_UNWIND_STEP_CONTINUE;
}

////////////////////////////////////////////////////////////////////////////////

BPF_MAP(heap, BPF_MAP_TYPE_PERCPU_ARRAY, u32, struct dwarf_unwind_context, 1);

////////////////////////////////////////////////////////////////////////////////

static ALWAYS_INLINE int dwarf_collect_stack(struct user_regs* regs, struct stack* stack) {
    u32 zero = 0;
    struct dwarf_unwind_context* ctx = bpf_map_lookup_elem(&heap, &zero);
    if (ctx == 0) {
        DWARF_TRACE("failed to load unwinder state from heap\n");
        return 0;
    }

    if (!dwarf_unwind_init(ctx, regs)) {
        DWARF_TRACE("failed to retrieve userspace registers\n");
        return 0;
    }

    stack->len = 0;

    int res = -1;
    for (int i = 0; i < DWARF_UNWIND_MAX_STACK_SIZE; ++i) {
        DWARF_TRACE("start iteration %d\n", i);
        enum dwarf_unwind_step_result res = dwarf_unwind_step(ctx, stack);
        switch (res) {
        case DWARF_UNWIND_STEP_FAILED:
            DWARF_TRACE("dwarf unwinding failed at step %d: %d\n", i, ctx->error);
            res = -1;
            goto done;
        case DWARF_UNWIND_STEP_FINISHED:
            DWARF_TRACE("dwarf unwinding finished\n");
            res = 0;
            goto done;
        case DWARF_UNWIND_STEP_CONTINUE:
            break;
        }
    }

    DWARF_TRACE("dwarf unwinding exhausted stack\n");

done:
    metric_add(METRIC_STACK_FRAME_DWARF_COUNT, stack->len - ctx->framepointers);
    metric_add(METRIC_STACK_FRAME_FP_COUNT, ctx->framepointers);
    metric_add(METRIC_STACK_FRAME_COUNT, stack->len);
    BPF_TRACE("found %u/%u frames using frame pointers\n", ctx->framepointers, stack->len);
    return res;
}
