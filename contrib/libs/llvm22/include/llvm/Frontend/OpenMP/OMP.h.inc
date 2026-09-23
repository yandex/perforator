#ifndef LLVM_OpenMP_INC
#define LLVM_OpenMP_INC

#include "llvm/ADT/ArrayRef.h"
#include "llvm/ADT/BitmaskEnum.h"
#include "llvm/ADT/Sequence.h"
#include "llvm/ADT/StringRef.h"
#include "llvm/Frontend/Directive/Spelling.h"
#include "llvm/Support/Compiler.h"
#include <cstddef>
#include <utility>

namespace llvm {

namespace omp {

LLVM_ENABLE_BITMASK_ENUMS_IN_NAMESPACE();

enum class Association {
  Block,
  First_ = Block,
  Declaration,
  Delimited,
  LoopNest,
  LoopSeq,
  None,
  Separating,
  Last_ = Separating,
};

static constexpr std::size_t Association_enumSize = 7;

enum class Category {
  Declarative,
  First_ = Declarative,
  Executable,
  Informational,
  Meta,
  Subsidiary,
  Utility,
  Last_ = Utility,
};

static constexpr std::size_t Category_enumSize = 6;

enum class SourceLanguage : uint32_t {
  C = 1U,
  Fortran = 2U,
  LLVM_MARK_AS_BITMASK_ENUM(/*LargestValue=*/Fortran)
};

static constexpr std::size_t SourceLanguage_enumSize = 2;

enum class Directive {
  OMPD_allocate,
  First_ = OMPD_allocate,
  OMPD_allocators,
  OMPD_assume,
  OMPD_assumes,
  OMPD_atomic,
  OMPD_barrier,
  OMPD_begin_assumes,
  OMPD_begin_declare_target,
  OMPD_begin_declare_variant,
  OMPD_cancel,
  OMPD_cancellation_point,
  OMPD_critical,
  OMPD_declare_induction,
  OMPD_declare_mapper,
  OMPD_declare_reduction,
  OMPD_declare_simd,
  OMPD_declare_target,
  OMPD_declare_variant,
  OMPD_depobj,
  OMPD_distribute,
  OMPD_distribute_parallel_do,
  OMPD_distribute_parallel_do_simd,
  OMPD_distribute_parallel_for,
  OMPD_distribute_parallel_for_simd,
  OMPD_distribute_simd,
  OMPD_do,
  OMPD_do_simd,
  OMPD_end_assumes,
  OMPD_end_declare_target,
  OMPD_end_declare_variant,
  OMPD_end_do,
  OMPD_end_do_simd,
  OMPD_end_scope,
  OMPD_end_sections,
  OMPD_end_single,
  OMPD_end_workdistribute,
  OMPD_end_workshare,
  OMPD_error,
  OMPD_flush,
  OMPD_for,
  OMPD_for_simd,
  OMPD_fuse,
  OMPD_groupprivate,
  OMPD_interchange,
  OMPD_masked_taskloop,
  OMPD_masked_taskloop_simd,
  OMPD_master,
  OMPD_master_taskloop,
  OMPD_master_taskloop_simd,
  OMPD_metadirective,
  OMPD_nothing,
  OMPD_ordered,
  OMPD_parallel,
  OMPD_parallel_do,
  OMPD_parallel_do_simd,
  OMPD_parallel_for,
  OMPD_parallel_for_simd,
  OMPD_parallel_masked,
  OMPD_parallel_masked_taskloop,
  OMPD_parallel_masked_taskloop_simd,
  OMPD_parallel_master,
  OMPD_parallel_master_taskloop,
  OMPD_parallel_master_taskloop_simd,
  OMPD_parallel_sections,
  OMPD_parallel_workshare,
  OMPD_requires,
  OMPD_reverse,
  OMPD_scan,
  OMPD_scope,
  OMPD_section,
  OMPD_sections,
  OMPD_simd,
  OMPD_single,
  OMPD_stripe,
  OMPD_target,
  OMPD_target_data,
  OMPD_target_enter_data,
  OMPD_target_exit_data,
  OMPD_target_parallel,
  OMPD_target_parallel_do,
  OMPD_target_parallel_do_simd,
  OMPD_target_parallel_for,
  OMPD_target_parallel_for_simd,
  OMPD_target_simd,
  OMPD_target_teams,
  OMPD_target_teams_distribute,
  OMPD_target_teams_distribute_parallel_do,
  OMPD_target_teams_distribute_parallel_do_simd,
  OMPD_target_teams_distribute_parallel_for,
  OMPD_target_teams_distribute_parallel_for_simd,
  OMPD_target_teams_distribute_simd,
  OMPD_target_teams_workdistribute,
  OMPD_target_update,
  OMPD_task,
  OMPD_taskgroup,
  OMPD_taskloop,
  OMPD_taskloop_simd,
  OMPD_taskwait,
  OMPD_taskyield,
  OMPD_taskgraph,
  OMPD_teams,
  OMPD_teams_distribute,
  OMPD_teams_distribute_parallel_do,
  OMPD_teams_distribute_parallel_do_simd,
  OMPD_teams_distribute_parallel_for,
  OMPD_teams_distribute_parallel_for_simd,
  OMPD_teams_distribute_simd,
  OMPD_teams_workdistribute,
  OMPD_threadprivate,
  OMPD_tile,
  OMPD_unknown,
  OMPD_unroll,
  OMPD_workdistribute,
  OMPD_workshare,
  OMPD_dispatch,
  OMPD_interop,
  OMPD_loop,
  OMPD_masked,
  OMPD_parallel_loop,
  OMPD_target_loop,
  OMPD_target_parallel_loop,
  OMPD_target_teams_loop,
  OMPD_teams_loop,
  Last_ = OMPD_teams_loop,
};

static constexpr std::size_t Directive_enumSize = 123;

constexpr auto OMPD_allocate = Directive::OMPD_allocate;
constexpr auto OMPD_allocators = Directive::OMPD_allocators;
constexpr auto OMPD_assume = Directive::OMPD_assume;
constexpr auto OMPD_assumes = Directive::OMPD_assumes;
constexpr auto OMPD_atomic = Directive::OMPD_atomic;
constexpr auto OMPD_barrier = Directive::OMPD_barrier;
constexpr auto OMPD_begin_assumes = Directive::OMPD_begin_assumes;
constexpr auto OMPD_begin_declare_target = Directive::OMPD_begin_declare_target;
constexpr auto OMPD_begin_declare_variant = Directive::OMPD_begin_declare_variant;
constexpr auto OMPD_cancel = Directive::OMPD_cancel;
constexpr auto OMPD_cancellation_point = Directive::OMPD_cancellation_point;
constexpr auto OMPD_critical = Directive::OMPD_critical;
constexpr auto OMPD_declare_induction = Directive::OMPD_declare_induction;
constexpr auto OMPD_declare_mapper = Directive::OMPD_declare_mapper;
constexpr auto OMPD_declare_reduction = Directive::OMPD_declare_reduction;
constexpr auto OMPD_declare_simd = Directive::OMPD_declare_simd;
constexpr auto OMPD_declare_target = Directive::OMPD_declare_target;
constexpr auto OMPD_declare_variant = Directive::OMPD_declare_variant;
constexpr auto OMPD_depobj = Directive::OMPD_depobj;
constexpr auto OMPD_distribute = Directive::OMPD_distribute;
constexpr auto OMPD_distribute_parallel_do = Directive::OMPD_distribute_parallel_do;
constexpr auto OMPD_distribute_parallel_do_simd = Directive::OMPD_distribute_parallel_do_simd;
constexpr auto OMPD_distribute_parallel_for = Directive::OMPD_distribute_parallel_for;
constexpr auto OMPD_distribute_parallel_for_simd = Directive::OMPD_distribute_parallel_for_simd;
constexpr auto OMPD_distribute_simd = Directive::OMPD_distribute_simd;
constexpr auto OMPD_do = Directive::OMPD_do;
constexpr auto OMPD_do_simd = Directive::OMPD_do_simd;
constexpr auto OMPD_end_assumes = Directive::OMPD_end_assumes;
constexpr auto OMPD_end_declare_target = Directive::OMPD_end_declare_target;
constexpr auto OMPD_end_declare_variant = Directive::OMPD_end_declare_variant;
constexpr auto OMPD_end_do = Directive::OMPD_end_do;
constexpr auto OMPD_end_do_simd = Directive::OMPD_end_do_simd;
constexpr auto OMPD_end_scope = Directive::OMPD_end_scope;
constexpr auto OMPD_end_sections = Directive::OMPD_end_sections;
constexpr auto OMPD_end_single = Directive::OMPD_end_single;
constexpr auto OMPD_end_workdistribute = Directive::OMPD_end_workdistribute;
constexpr auto OMPD_end_workshare = Directive::OMPD_end_workshare;
constexpr auto OMPD_error = Directive::OMPD_error;
constexpr auto OMPD_flush = Directive::OMPD_flush;
constexpr auto OMPD_for = Directive::OMPD_for;
constexpr auto OMPD_for_simd = Directive::OMPD_for_simd;
constexpr auto OMPD_fuse = Directive::OMPD_fuse;
constexpr auto OMPD_groupprivate = Directive::OMPD_groupprivate;
constexpr auto OMPD_interchange = Directive::OMPD_interchange;
constexpr auto OMPD_masked_taskloop = Directive::OMPD_masked_taskloop;
constexpr auto OMPD_masked_taskloop_simd = Directive::OMPD_masked_taskloop_simd;
constexpr auto OMPD_master = Directive::OMPD_master;
constexpr auto OMPD_master_taskloop = Directive::OMPD_master_taskloop;
constexpr auto OMPD_master_taskloop_simd = Directive::OMPD_master_taskloop_simd;
constexpr auto OMPD_metadirective = Directive::OMPD_metadirective;
constexpr auto OMPD_nothing = Directive::OMPD_nothing;
constexpr auto OMPD_ordered = Directive::OMPD_ordered;
constexpr auto OMPD_parallel = Directive::OMPD_parallel;
constexpr auto OMPD_parallel_do = Directive::OMPD_parallel_do;
constexpr auto OMPD_parallel_do_simd = Directive::OMPD_parallel_do_simd;
constexpr auto OMPD_parallel_for = Directive::OMPD_parallel_for;
constexpr auto OMPD_parallel_for_simd = Directive::OMPD_parallel_for_simd;
constexpr auto OMPD_parallel_masked = Directive::OMPD_parallel_masked;
constexpr auto OMPD_parallel_masked_taskloop = Directive::OMPD_parallel_masked_taskloop;
constexpr auto OMPD_parallel_masked_taskloop_simd = Directive::OMPD_parallel_masked_taskloop_simd;
constexpr auto OMPD_parallel_master = Directive::OMPD_parallel_master;
constexpr auto OMPD_parallel_master_taskloop = Directive::OMPD_parallel_master_taskloop;
constexpr auto OMPD_parallel_master_taskloop_simd = Directive::OMPD_parallel_master_taskloop_simd;
constexpr auto OMPD_parallel_sections = Directive::OMPD_parallel_sections;
constexpr auto OMPD_parallel_workshare = Directive::OMPD_parallel_workshare;
constexpr auto OMPD_requires = Directive::OMPD_requires;
constexpr auto OMPD_reverse = Directive::OMPD_reverse;
constexpr auto OMPD_scan = Directive::OMPD_scan;
constexpr auto OMPD_scope = Directive::OMPD_scope;
constexpr auto OMPD_section = Directive::OMPD_section;
constexpr auto OMPD_sections = Directive::OMPD_sections;
constexpr auto OMPD_simd = Directive::OMPD_simd;
constexpr auto OMPD_single = Directive::OMPD_single;
constexpr auto OMPD_stripe = Directive::OMPD_stripe;
constexpr auto OMPD_target = Directive::OMPD_target;
constexpr auto OMPD_target_data = Directive::OMPD_target_data;
constexpr auto OMPD_target_enter_data = Directive::OMPD_target_enter_data;
constexpr auto OMPD_target_exit_data = Directive::OMPD_target_exit_data;
constexpr auto OMPD_target_parallel = Directive::OMPD_target_parallel;
constexpr auto OMPD_target_parallel_do = Directive::OMPD_target_parallel_do;
constexpr auto OMPD_target_parallel_do_simd = Directive::OMPD_target_parallel_do_simd;
constexpr auto OMPD_target_parallel_for = Directive::OMPD_target_parallel_for;
constexpr auto OMPD_target_parallel_for_simd = Directive::OMPD_target_parallel_for_simd;
constexpr auto OMPD_target_simd = Directive::OMPD_target_simd;
constexpr auto OMPD_target_teams = Directive::OMPD_target_teams;
constexpr auto OMPD_target_teams_distribute = Directive::OMPD_target_teams_distribute;
constexpr auto OMPD_target_teams_distribute_parallel_do = Directive::OMPD_target_teams_distribute_parallel_do;
constexpr auto OMPD_target_teams_distribute_parallel_do_simd = Directive::OMPD_target_teams_distribute_parallel_do_simd;
constexpr auto OMPD_target_teams_distribute_parallel_for = Directive::OMPD_target_teams_distribute_parallel_for;
constexpr auto OMPD_target_teams_distribute_parallel_for_simd = Directive::OMPD_target_teams_distribute_parallel_for_simd;
constexpr auto OMPD_target_teams_distribute_simd = Directive::OMPD_target_teams_distribute_simd;
constexpr auto OMPD_target_teams_workdistribute = Directive::OMPD_target_teams_workdistribute;
constexpr auto OMPD_target_update = Directive::OMPD_target_update;
constexpr auto OMPD_task = Directive::OMPD_task;
constexpr auto OMPD_taskgroup = Directive::OMPD_taskgroup;
constexpr auto OMPD_taskloop = Directive::OMPD_taskloop;
constexpr auto OMPD_taskloop_simd = Directive::OMPD_taskloop_simd;
constexpr auto OMPD_taskwait = Directive::OMPD_taskwait;
constexpr auto OMPD_taskyield = Directive::OMPD_taskyield;
constexpr auto OMPD_taskgraph = Directive::OMPD_taskgraph;
constexpr auto OMPD_teams = Directive::OMPD_teams;
constexpr auto OMPD_teams_distribute = Directive::OMPD_teams_distribute;
constexpr auto OMPD_teams_distribute_parallel_do = Directive::OMPD_teams_distribute_parallel_do;
constexpr auto OMPD_teams_distribute_parallel_do_simd = Directive::OMPD_teams_distribute_parallel_do_simd;
constexpr auto OMPD_teams_distribute_parallel_for = Directive::OMPD_teams_distribute_parallel_for;
constexpr auto OMPD_teams_distribute_parallel_for_simd = Directive::OMPD_teams_distribute_parallel_for_simd;
constexpr auto OMPD_teams_distribute_simd = Directive::OMPD_teams_distribute_simd;
constexpr auto OMPD_teams_workdistribute = Directive::OMPD_teams_workdistribute;
constexpr auto OMPD_threadprivate = Directive::OMPD_threadprivate;
constexpr auto OMPD_tile = Directive::OMPD_tile;
constexpr auto OMPD_unknown = Directive::OMPD_unknown;
constexpr auto OMPD_unroll = Directive::OMPD_unroll;
constexpr auto OMPD_workdistribute = Directive::OMPD_workdistribute;
constexpr auto OMPD_workshare = Directive::OMPD_workshare;
constexpr auto OMPD_dispatch = Directive::OMPD_dispatch;
constexpr auto OMPD_interop = Directive::OMPD_interop;
constexpr auto OMPD_loop = Directive::OMPD_loop;
constexpr auto OMPD_masked = Directive::OMPD_masked;
constexpr auto OMPD_parallel_loop = Directive::OMPD_parallel_loop;
constexpr auto OMPD_target_loop = Directive::OMPD_target_loop;
constexpr auto OMPD_target_parallel_loop = Directive::OMPD_target_parallel_loop;
constexpr auto OMPD_target_teams_loop = Directive::OMPD_target_teams_loop;
constexpr auto OMPD_teams_loop = Directive::OMPD_teams_loop;

enum class Clause {
  OMPC_absent,
  First_ = OMPC_absent,
  OMPC_acq_rel,
  OMPC_acquire,
  OMPC_adjust_args,
  OMPC_affinity,
  OMPC_align,
  OMPC_aligned,
  OMPC_allocate,
  OMPC_allocator,
  OMPC_append_args,
  OMPC_apply,
  OMPC_at,
  OMPC_atomic_default_mem_order,
  OMPC_bind,
  OMPC_cancellation_construct_type,
  OMPC_capture,
  OMPC_collapse,
  OMPC_collector,
  OMPC_combiner,
  OMPC_compare,
  OMPC_contains,
  OMPC_copyprivate,
  OMPC_copyin,
  OMPC_counts,
  OMPC_default,
  OMPC_defaultmap,
  OMPC_depend,
  OMPC_depobj,
  OMPC_destroy,
  OMPC_detach,
  OMPC_device,
  OMPC_device_safesync,
  OMPC_device_type,
  OMPC_dist_schedule,
  OMPC_doacross,
  OMPC_dyn_groupprivate,
  OMPC_dynamic_allocators,
  OMPC_enter,
  OMPC_exclusive,
  OMPC_fail,
  OMPC_filter,
  OMPC_final,
  OMPC_firstprivate,
  OMPC_flush,
  OMPC_from,
  OMPC_full,
  OMPC_grainsize,
  OMPC_graph_id,
  OMPC_graph_reset,
  OMPC_groupprivate,
  OMPC_has_device_addr,
  OMPC_hint,
  OMPC_holds,
  OMPC_if,
  OMPC_in_reduction,
  OMPC_inbranch,
  OMPC_inclusive,
  OMPC_indirect,
  OMPC_induction,
  OMPC_inductor,
  OMPC_init,
  OMPC_init_complete,
  OMPC_initializer,
  OMPC_interop,
  OMPC_is_device_ptr,
  OMPC_lastprivate,
  OMPC_linear,
  OMPC_link,
  OMPC_local,
  OMPC_looprange,
  OMPC_map,
  OMPC_match,
  OMPC_memory_order,
  OMPC_memscope,
  OMPC_mergeable,
  OMPC_message,
  OMPC_nogroup,
  OMPC_no_openmp,
  OMPC_no_openmp_constructs,
  OMPC_no_openmp_routines,
  OMPC_no_parallelism,
  OMPC_nowait,
  OMPC_nocontext,
  OMPC_nontemporal,
  OMPC_notinbranch,
  OMPC_novariants,
  OMPC_num_tasks,
  OMPC_num_teams,
  OMPC_num_threads,
  OMPC_ompx_attribute,
  OMPC_ompx_bare,
  OMPC_ompx_dyn_cgroup_mem,
  OMPC_order,
  OMPC_ordered,
  OMPC_otherwise,
  OMPC_partial,
  OMPC_permutation,
  OMPC_priority,
  OMPC_private,
  OMPC_proc_bind,
  OMPC_read,
  OMPC_reduction,
  OMPC_relaxed,
  OMPC_release,
  OMPC_replayable,
  OMPC_reverse_offload,
  OMPC_safelen,
  OMPC_safesync,
  OMPC_schedule,
  OMPC_self_maps,
  OMPC_seq_cst,
  OMPC_severity,
  OMPC_shared,
  OMPC_simd,
  OMPC_simdlen,
  OMPC_sizes,
  OMPC_task_reduction,
  OMPC_thread_limit,
  OMPC_threadprivate,
  OMPC_threads,
  OMPC_threadset,
  OMPC_to,
  OMPC_transparent,
  OMPC_unified_address,
  OMPC_unified_shared_memory,
  OMPC_uniform,
  OMPC_unknown,
  OMPC_untied,
  OMPC_update,
  OMPC_use,
  OMPC_use_device_addr,
  OMPC_use_device_ptr,
  OMPC_uses_allocators,
  OMPC_weak,
  OMPC_when,
  OMPC_write,
  Last_ = OMPC_write,
};

static constexpr std::size_t Clause_enumSize = 136;

constexpr auto OMPC_absent = Clause::OMPC_absent;
constexpr auto OMPC_acq_rel = Clause::OMPC_acq_rel;
constexpr auto OMPC_acquire = Clause::OMPC_acquire;
constexpr auto OMPC_adjust_args = Clause::OMPC_adjust_args;
constexpr auto OMPC_affinity = Clause::OMPC_affinity;
constexpr auto OMPC_align = Clause::OMPC_align;
constexpr auto OMPC_aligned = Clause::OMPC_aligned;
constexpr auto OMPC_allocate = Clause::OMPC_allocate;
constexpr auto OMPC_allocator = Clause::OMPC_allocator;
constexpr auto OMPC_append_args = Clause::OMPC_append_args;
constexpr auto OMPC_apply = Clause::OMPC_apply;
constexpr auto OMPC_at = Clause::OMPC_at;
constexpr auto OMPC_atomic_default_mem_order = Clause::OMPC_atomic_default_mem_order;
constexpr auto OMPC_bind = Clause::OMPC_bind;
constexpr auto OMPC_cancellation_construct_type = Clause::OMPC_cancellation_construct_type;
constexpr auto OMPC_capture = Clause::OMPC_capture;
constexpr auto OMPC_collapse = Clause::OMPC_collapse;
constexpr auto OMPC_collector = Clause::OMPC_collector;
constexpr auto OMPC_combiner = Clause::OMPC_combiner;
constexpr auto OMPC_compare = Clause::OMPC_compare;
constexpr auto OMPC_contains = Clause::OMPC_contains;
constexpr auto OMPC_copyprivate = Clause::OMPC_copyprivate;
constexpr auto OMPC_copyin = Clause::OMPC_copyin;
constexpr auto OMPC_counts = Clause::OMPC_counts;
constexpr auto OMPC_default = Clause::OMPC_default;
constexpr auto OMPC_defaultmap = Clause::OMPC_defaultmap;
constexpr auto OMPC_depend = Clause::OMPC_depend;
constexpr auto OMPC_depobj = Clause::OMPC_depobj;
constexpr auto OMPC_destroy = Clause::OMPC_destroy;
constexpr auto OMPC_detach = Clause::OMPC_detach;
constexpr auto OMPC_device = Clause::OMPC_device;
constexpr auto OMPC_device_safesync = Clause::OMPC_device_safesync;
constexpr auto OMPC_device_type = Clause::OMPC_device_type;
constexpr auto OMPC_dist_schedule = Clause::OMPC_dist_schedule;
constexpr auto OMPC_doacross = Clause::OMPC_doacross;
constexpr auto OMPC_dyn_groupprivate = Clause::OMPC_dyn_groupprivate;
constexpr auto OMPC_dynamic_allocators = Clause::OMPC_dynamic_allocators;
constexpr auto OMPC_enter = Clause::OMPC_enter;
constexpr auto OMPC_exclusive = Clause::OMPC_exclusive;
constexpr auto OMPC_fail = Clause::OMPC_fail;
constexpr auto OMPC_filter = Clause::OMPC_filter;
constexpr auto OMPC_final = Clause::OMPC_final;
constexpr auto OMPC_firstprivate = Clause::OMPC_firstprivate;
constexpr auto OMPC_flush = Clause::OMPC_flush;
constexpr auto OMPC_from = Clause::OMPC_from;
constexpr auto OMPC_full = Clause::OMPC_full;
constexpr auto OMPC_grainsize = Clause::OMPC_grainsize;
constexpr auto OMPC_graph_id = Clause::OMPC_graph_id;
constexpr auto OMPC_graph_reset = Clause::OMPC_graph_reset;
constexpr auto OMPC_groupprivate = Clause::OMPC_groupprivate;
constexpr auto OMPC_has_device_addr = Clause::OMPC_has_device_addr;
constexpr auto OMPC_hint = Clause::OMPC_hint;
constexpr auto OMPC_holds = Clause::OMPC_holds;
constexpr auto OMPC_if = Clause::OMPC_if;
constexpr auto OMPC_in_reduction = Clause::OMPC_in_reduction;
constexpr auto OMPC_inbranch = Clause::OMPC_inbranch;
constexpr auto OMPC_inclusive = Clause::OMPC_inclusive;
constexpr auto OMPC_indirect = Clause::OMPC_indirect;
constexpr auto OMPC_induction = Clause::OMPC_induction;
constexpr auto OMPC_inductor = Clause::OMPC_inductor;
constexpr auto OMPC_init = Clause::OMPC_init;
constexpr auto OMPC_init_complete = Clause::OMPC_init_complete;
constexpr auto OMPC_initializer = Clause::OMPC_initializer;
constexpr auto OMPC_interop = Clause::OMPC_interop;
constexpr auto OMPC_is_device_ptr = Clause::OMPC_is_device_ptr;
constexpr auto OMPC_lastprivate = Clause::OMPC_lastprivate;
constexpr auto OMPC_linear = Clause::OMPC_linear;
constexpr auto OMPC_link = Clause::OMPC_link;
constexpr auto OMPC_local = Clause::OMPC_local;
constexpr auto OMPC_looprange = Clause::OMPC_looprange;
constexpr auto OMPC_map = Clause::OMPC_map;
constexpr auto OMPC_match = Clause::OMPC_match;
constexpr auto OMPC_memory_order = Clause::OMPC_memory_order;
constexpr auto OMPC_memscope = Clause::OMPC_memscope;
constexpr auto OMPC_mergeable = Clause::OMPC_mergeable;
constexpr auto OMPC_message = Clause::OMPC_message;
constexpr auto OMPC_nogroup = Clause::OMPC_nogroup;
constexpr auto OMPC_no_openmp = Clause::OMPC_no_openmp;
constexpr auto OMPC_no_openmp_constructs = Clause::OMPC_no_openmp_constructs;
constexpr auto OMPC_no_openmp_routines = Clause::OMPC_no_openmp_routines;
constexpr auto OMPC_no_parallelism = Clause::OMPC_no_parallelism;
constexpr auto OMPC_nowait = Clause::OMPC_nowait;
constexpr auto OMPC_nocontext = Clause::OMPC_nocontext;
constexpr auto OMPC_nontemporal = Clause::OMPC_nontemporal;
constexpr auto OMPC_notinbranch = Clause::OMPC_notinbranch;
constexpr auto OMPC_novariants = Clause::OMPC_novariants;
constexpr auto OMPC_num_tasks = Clause::OMPC_num_tasks;
constexpr auto OMPC_num_teams = Clause::OMPC_num_teams;
constexpr auto OMPC_num_threads = Clause::OMPC_num_threads;
constexpr auto OMPC_ompx_attribute = Clause::OMPC_ompx_attribute;
constexpr auto OMPC_ompx_bare = Clause::OMPC_ompx_bare;
constexpr auto OMPC_ompx_dyn_cgroup_mem = Clause::OMPC_ompx_dyn_cgroup_mem;
constexpr auto OMPC_order = Clause::OMPC_order;
constexpr auto OMPC_ordered = Clause::OMPC_ordered;
constexpr auto OMPC_otherwise = Clause::OMPC_otherwise;
constexpr auto OMPC_partial = Clause::OMPC_partial;
constexpr auto OMPC_permutation = Clause::OMPC_permutation;
constexpr auto OMPC_priority = Clause::OMPC_priority;
constexpr auto OMPC_private = Clause::OMPC_private;
constexpr auto OMPC_proc_bind = Clause::OMPC_proc_bind;
constexpr auto OMPC_read = Clause::OMPC_read;
constexpr auto OMPC_reduction = Clause::OMPC_reduction;
constexpr auto OMPC_relaxed = Clause::OMPC_relaxed;
constexpr auto OMPC_release = Clause::OMPC_release;
constexpr auto OMPC_replayable = Clause::OMPC_replayable;
constexpr auto OMPC_reverse_offload = Clause::OMPC_reverse_offload;
constexpr auto OMPC_safelen = Clause::OMPC_safelen;
constexpr auto OMPC_safesync = Clause::OMPC_safesync;
constexpr auto OMPC_schedule = Clause::OMPC_schedule;
constexpr auto OMPC_self_maps = Clause::OMPC_self_maps;
constexpr auto OMPC_seq_cst = Clause::OMPC_seq_cst;
constexpr auto OMPC_severity = Clause::OMPC_severity;
constexpr auto OMPC_shared = Clause::OMPC_shared;
constexpr auto OMPC_simd = Clause::OMPC_simd;
constexpr auto OMPC_simdlen = Clause::OMPC_simdlen;
constexpr auto OMPC_sizes = Clause::OMPC_sizes;
constexpr auto OMPC_task_reduction = Clause::OMPC_task_reduction;
constexpr auto OMPC_thread_limit = Clause::OMPC_thread_limit;
constexpr auto OMPC_threadprivate = Clause::OMPC_threadprivate;
constexpr auto OMPC_threads = Clause::OMPC_threads;
constexpr auto OMPC_threadset = Clause::OMPC_threadset;
constexpr auto OMPC_to = Clause::OMPC_to;
constexpr auto OMPC_transparent = Clause::OMPC_transparent;
constexpr auto OMPC_unified_address = Clause::OMPC_unified_address;
constexpr auto OMPC_unified_shared_memory = Clause::OMPC_unified_shared_memory;
constexpr auto OMPC_uniform = Clause::OMPC_uniform;
constexpr auto OMPC_unknown = Clause::OMPC_unknown;
constexpr auto OMPC_untied = Clause::OMPC_untied;
constexpr auto OMPC_update = Clause::OMPC_update;
constexpr auto OMPC_use = Clause::OMPC_use;
constexpr auto OMPC_use_device_addr = Clause::OMPC_use_device_addr;
constexpr auto OMPC_use_device_ptr = Clause::OMPC_use_device_ptr;
constexpr auto OMPC_uses_allocators = Clause::OMPC_uses_allocators;
constexpr auto OMPC_weak = Clause::OMPC_weak;
constexpr auto OMPC_when = Clause::OMPC_when;
constexpr auto OMPC_write = Clause::OMPC_write;

enum class BindKind {
  OMP_BIND_parallel=1,
  OMP_BIND_teams=2,
  OMP_BIND_thread=3,
};

constexpr auto OMP_BIND_parallel = BindKind::OMP_BIND_parallel;
constexpr auto OMP_BIND_teams = BindKind::OMP_BIND_teams;
constexpr auto OMP_BIND_thread = BindKind::OMP_BIND_thread;

enum class CancellationConstructType {
  OMP_CANCELLATION_CONSTRUCT_Parallel=1,
  OMP_CANCELLATION_CONSTRUCT_Loop=2,
  OMP_CANCELLATION_CONSTRUCT_Sections=3,
  OMP_CANCELLATION_CONSTRUCT_Taskgroup=4,
  OMP_CANCELLATION_CONSTRUCT_None=5,
};

constexpr auto OMP_CANCELLATION_CONSTRUCT_Parallel = CancellationConstructType::OMP_CANCELLATION_CONSTRUCT_Parallel;
constexpr auto OMP_CANCELLATION_CONSTRUCT_Loop = CancellationConstructType::OMP_CANCELLATION_CONSTRUCT_Loop;
constexpr auto OMP_CANCELLATION_CONSTRUCT_Sections = CancellationConstructType::OMP_CANCELLATION_CONSTRUCT_Sections;
constexpr auto OMP_CANCELLATION_CONSTRUCT_Taskgroup = CancellationConstructType::OMP_CANCELLATION_CONSTRUCT_Taskgroup;
constexpr auto OMP_CANCELLATION_CONSTRUCT_None = CancellationConstructType::OMP_CANCELLATION_CONSTRUCT_None;

enum class GrainsizeType {
  OMP_GRAINSIZE_Strict=1,
  OMP_GRAINSIZE_Unknown=2,
};

constexpr auto OMP_GRAINSIZE_Strict = GrainsizeType::OMP_GRAINSIZE_Strict;
constexpr auto OMP_GRAINSIZE_Unknown = GrainsizeType::OMP_GRAINSIZE_Unknown;

enum class MemoryOrderKind {
  OMP_MEMORY_ORDER_SeqCst=1,
  OMP_MEMORY_ORDER_AcqRel=2,
  OMP_MEMORY_ORDER_Acquire=3,
  OMP_MEMORY_ORDER_Release=4,
  OMP_MEMORY_ORDER_Relaxed=5,
  OMP_MEMORY_ORDER_Default=6,
};

constexpr auto OMP_MEMORY_ORDER_SeqCst = MemoryOrderKind::OMP_MEMORY_ORDER_SeqCst;
constexpr auto OMP_MEMORY_ORDER_AcqRel = MemoryOrderKind::OMP_MEMORY_ORDER_AcqRel;
constexpr auto OMP_MEMORY_ORDER_Acquire = MemoryOrderKind::OMP_MEMORY_ORDER_Acquire;
constexpr auto OMP_MEMORY_ORDER_Release = MemoryOrderKind::OMP_MEMORY_ORDER_Release;
constexpr auto OMP_MEMORY_ORDER_Relaxed = MemoryOrderKind::OMP_MEMORY_ORDER_Relaxed;
constexpr auto OMP_MEMORY_ORDER_Default = MemoryOrderKind::OMP_MEMORY_ORDER_Default;

enum class NumTasksType {
  OMP_NUMTASKS_Strict=1,
  OMP_NUMTASKS_Unknown=2,
};

constexpr auto OMP_NUMTASKS_Strict = NumTasksType::OMP_NUMTASKS_Strict;
constexpr auto OMP_NUMTASKS_Unknown = NumTasksType::OMP_NUMTASKS_Unknown;

enum class NumThreadsType {
  OMP_NUMTHREADS_Strict=1,
  OMP_NUMTHREADS_Unknown=2,
};

constexpr auto OMP_NUMTHREADS_Strict = NumThreadsType::OMP_NUMTHREADS_Strict;
constexpr auto OMP_NUMTHREADS_Unknown = NumThreadsType::OMP_NUMTHREADS_Unknown;

enum class OrderKind {
  OMP_ORDER_unknown=2,
  OMP_ORDER_concurrent=1,
};

constexpr auto OMP_ORDER_unknown = OrderKind::OMP_ORDER_unknown;
constexpr auto OMP_ORDER_concurrent = OrderKind::OMP_ORDER_concurrent;

enum class ProcBindKind {
  OMP_PROC_BIND_primary=5,
  OMP_PROC_BIND_master=2,
  OMP_PROC_BIND_close=3,
  OMP_PROC_BIND_spread=4,
  OMP_PROC_BIND_default=6,
  OMP_PROC_BIND_unknown=7,
};

constexpr auto OMP_PROC_BIND_primary = ProcBindKind::OMP_PROC_BIND_primary;
constexpr auto OMP_PROC_BIND_master = ProcBindKind::OMP_PROC_BIND_master;
constexpr auto OMP_PROC_BIND_close = ProcBindKind::OMP_PROC_BIND_close;
constexpr auto OMP_PROC_BIND_spread = ProcBindKind::OMP_PROC_BIND_spread;
constexpr auto OMP_PROC_BIND_default = ProcBindKind::OMP_PROC_BIND_default;
constexpr auto OMP_PROC_BIND_unknown = ProcBindKind::OMP_PROC_BIND_unknown;

enum class ScheduleKind {
  OMP_SCHEDULE_Static=2,
  OMP_SCHEDULE_Dynamic=3,
  OMP_SCHEDULE_Guided=4,
  OMP_SCHEDULE_Auto=5,
  OMP_SCHEDULE_Runtime=6,
  OMP_SCHEDULE_Distribute=7,
  OMP_SCHEDULE_Default=8,
};

constexpr auto OMP_SCHEDULE_Static = ScheduleKind::OMP_SCHEDULE_Static;
constexpr auto OMP_SCHEDULE_Dynamic = ScheduleKind::OMP_SCHEDULE_Dynamic;
constexpr auto OMP_SCHEDULE_Guided = ScheduleKind::OMP_SCHEDULE_Guided;
constexpr auto OMP_SCHEDULE_Auto = ScheduleKind::OMP_SCHEDULE_Auto;
constexpr auto OMP_SCHEDULE_Runtime = ScheduleKind::OMP_SCHEDULE_Runtime;
constexpr auto OMP_SCHEDULE_Distribute = ScheduleKind::OMP_SCHEDULE_Distribute;
constexpr auto OMP_SCHEDULE_Default = ScheduleKind::OMP_SCHEDULE_Default;

// Enumeration helper functions
LLVM_ABI std::pair<Directive, directive::VersionRange> getOpenMPDirectiveKindAndVersions(StringRef Str);
inline Directive getOpenMPDirectiveKind(StringRef Str) {
  return getOpenMPDirectiveKindAndVersions(Str).first;
}

LLVM_ABI StringRef getOpenMPDirectiveName(Directive D, unsigned Ver = 0);

LLVM_ABI std::pair<Clause, directive::VersionRange> getOpenMPClauseKindAndVersions(StringRef Str);

inline Clause getOpenMPClauseKind(StringRef Str) {
  return getOpenMPClauseKindAndVersions(Str).first;
}

LLVM_ABI StringRef getOpenMPClauseName(Clause C, unsigned Ver = 0);

/// Return true if \p C is a valid clause for \p D in version \p Version.
LLVM_ABI bool isAllowedClauseForDirective(Directive D, Clause C, unsigned Version);

constexpr std::size_t getMaxLeafCount() { return 6; }
LLVM_ABI Association getDirectiveAssociation(Directive D);
LLVM_ABI Category getDirectiveCategory(Directive D);
LLVM_ABI SourceLanguage getDirectiveLanguages(Directive D);
LLVM_ABI BindKind getBindKind(StringRef Str);
LLVM_ABI StringRef getOpenMPBindKindName(BindKind x);
LLVM_ABI CancellationConstructType getCancellationConstructType(StringRef Str);
LLVM_ABI StringRef getOpenMPCancellationConstructTypeName(CancellationConstructType x);
LLVM_ABI GrainsizeType getGrainsizeType(StringRef Str);
LLVM_ABI StringRef getOpenMPGrainsizeTypeName(GrainsizeType x);
LLVM_ABI MemoryOrderKind getMemoryOrderKind(StringRef Str);
LLVM_ABI StringRef getOpenMPMemoryOrderKindName(MemoryOrderKind x);
LLVM_ABI NumTasksType getNumTasksType(StringRef Str);
LLVM_ABI StringRef getOpenMPNumTasksTypeName(NumTasksType x);
LLVM_ABI NumThreadsType getNumThreadsType(StringRef Str);
LLVM_ABI StringRef getOpenMPNumThreadsTypeName(NumThreadsType x);
LLVM_ABI OrderKind getOrderKind(StringRef Str);
LLVM_ABI StringRef getOpenMPOrderKindName(OrderKind x);
LLVM_ABI ProcBindKind getProcBindKind(StringRef Str);
LLVM_ABI StringRef getOpenMPProcBindKindName(ProcBindKind x);
LLVM_ABI ScheduleKind getScheduleKind(StringRef Str);
LLVM_ABI StringRef getOpenMPScheduleKindName(ScheduleKind x);

} // namespace omp

template <> struct enum_iteration_traits<omp::Association> {
  static constexpr bool is_iterable = true;
};

template <> struct enum_iteration_traits<omp::Category> {
  static constexpr bool is_iterable = true;
};

template <> struct enum_iteration_traits<omp::Directive> {
  static constexpr bool is_iterable = true;
};

template <> struct enum_iteration_traits<omp::Clause> {
  static constexpr bool is_iterable = true;
};

} // namespace llvm

#endif // LLVM_OpenMP_INC

