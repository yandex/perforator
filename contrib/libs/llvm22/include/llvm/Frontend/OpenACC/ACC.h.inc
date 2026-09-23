#ifndef LLVM_OpenACC_INC
#define LLVM_OpenACC_INC

#include "llvm/ADT/ArrayRef.h"
#include "llvm/ADT/BitmaskEnum.h"
#include "llvm/ADT/Sequence.h"
#include "llvm/ADT/StringRef.h"
#include "llvm/Frontend/Directive/Spelling.h"
#include "llvm/Support/Compiler.h"
#include <cstddef>
#include <utility>

namespace llvm {

namespace acc {

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
  ACCD_atomic,
  First_ = ACCD_atomic,
  ACCD_cache,
  ACCD_data,
  ACCD_declare,
  ACCD_enter_data,
  ACCD_exit_data,
  ACCD_host_data,
  ACCD_init,
  ACCD_kernels,
  ACCD_kernels_loop,
  ACCD_loop,
  ACCD_parallel,
  ACCD_parallel_loop,
  ACCD_routine,
  ACCD_serial,
  ACCD_serial_loop,
  ACCD_set,
  ACCD_shutdown,
  ACCD_unknown,
  ACCD_update,
  ACCD_wait,
  Last_ = ACCD_wait,
};

static constexpr std::size_t Directive_enumSize = 21;

constexpr auto ACCD_atomic = Directive::ACCD_atomic;
constexpr auto ACCD_cache = Directive::ACCD_cache;
constexpr auto ACCD_data = Directive::ACCD_data;
constexpr auto ACCD_declare = Directive::ACCD_declare;
constexpr auto ACCD_enter_data = Directive::ACCD_enter_data;
constexpr auto ACCD_exit_data = Directive::ACCD_exit_data;
constexpr auto ACCD_host_data = Directive::ACCD_host_data;
constexpr auto ACCD_init = Directive::ACCD_init;
constexpr auto ACCD_kernels = Directive::ACCD_kernels;
constexpr auto ACCD_kernels_loop = Directive::ACCD_kernels_loop;
constexpr auto ACCD_loop = Directive::ACCD_loop;
constexpr auto ACCD_parallel = Directive::ACCD_parallel;
constexpr auto ACCD_parallel_loop = Directive::ACCD_parallel_loop;
constexpr auto ACCD_routine = Directive::ACCD_routine;
constexpr auto ACCD_serial = Directive::ACCD_serial;
constexpr auto ACCD_serial_loop = Directive::ACCD_serial_loop;
constexpr auto ACCD_set = Directive::ACCD_set;
constexpr auto ACCD_shutdown = Directive::ACCD_shutdown;
constexpr auto ACCD_unknown = Directive::ACCD_unknown;
constexpr auto ACCD_update = Directive::ACCD_update;
constexpr auto ACCD_wait = Directive::ACCD_wait;

enum class Clause {
  ACCC_async,
  First_ = ACCC_async,
  ACCC_attach,
  ACCC_auto,
  ACCC_bind,
  ACCC_capture,
  ACCC_collapse,
  ACCC_copy,
  ACCC_copyin,
  ACCC_copyout,
  ACCC_create,
  ACCC_default,
  ACCC_default_async,
  ACCC_delete,
  ACCC_detach,
  ACCC_device,
  ACCC_device_num,
  ACCC_deviceptr,
  ACCC_device_resident,
  ACCC_device_type,
  ACCC_finalize,
  ACCC_firstprivate,
  ACCC_gang,
  ACCC_host,
  ACCC_if,
  ACCC_if_present,
  ACCC_independent,
  ACCC_link,
  ACCC_no_create,
  ACCC_nohost,
  ACCC_num_gangs,
  ACCC_num_workers,
  ACCC_present,
  ACCC_private,
  ACCC_read,
  ACCC_reduction,
  ACCC_self,
  ACCC_seq,
  ACCC_shortloop,
  ACCC_tile,
  ACCC_unknown,
  ACCC_use_device,
  ACCC_vector,
  ACCC_vector_length,
  ACCC_wait,
  ACCC_worker,
  ACCC_write,
  Last_ = ACCC_write,
};

static constexpr std::size_t Clause_enumSize = 46;

constexpr auto ACCC_async = Clause::ACCC_async;
constexpr auto ACCC_attach = Clause::ACCC_attach;
constexpr auto ACCC_auto = Clause::ACCC_auto;
constexpr auto ACCC_bind = Clause::ACCC_bind;
constexpr auto ACCC_capture = Clause::ACCC_capture;
constexpr auto ACCC_collapse = Clause::ACCC_collapse;
constexpr auto ACCC_copy = Clause::ACCC_copy;
constexpr auto ACCC_copyin = Clause::ACCC_copyin;
constexpr auto ACCC_copyout = Clause::ACCC_copyout;
constexpr auto ACCC_create = Clause::ACCC_create;
constexpr auto ACCC_default = Clause::ACCC_default;
constexpr auto ACCC_default_async = Clause::ACCC_default_async;
constexpr auto ACCC_delete = Clause::ACCC_delete;
constexpr auto ACCC_detach = Clause::ACCC_detach;
constexpr auto ACCC_device = Clause::ACCC_device;
constexpr auto ACCC_device_num = Clause::ACCC_device_num;
constexpr auto ACCC_deviceptr = Clause::ACCC_deviceptr;
constexpr auto ACCC_device_resident = Clause::ACCC_device_resident;
constexpr auto ACCC_device_type = Clause::ACCC_device_type;
constexpr auto ACCC_finalize = Clause::ACCC_finalize;
constexpr auto ACCC_firstprivate = Clause::ACCC_firstprivate;
constexpr auto ACCC_gang = Clause::ACCC_gang;
constexpr auto ACCC_host = Clause::ACCC_host;
constexpr auto ACCC_if = Clause::ACCC_if;
constexpr auto ACCC_if_present = Clause::ACCC_if_present;
constexpr auto ACCC_independent = Clause::ACCC_independent;
constexpr auto ACCC_link = Clause::ACCC_link;
constexpr auto ACCC_no_create = Clause::ACCC_no_create;
constexpr auto ACCC_nohost = Clause::ACCC_nohost;
constexpr auto ACCC_num_gangs = Clause::ACCC_num_gangs;
constexpr auto ACCC_num_workers = Clause::ACCC_num_workers;
constexpr auto ACCC_present = Clause::ACCC_present;
constexpr auto ACCC_private = Clause::ACCC_private;
constexpr auto ACCC_read = Clause::ACCC_read;
constexpr auto ACCC_reduction = Clause::ACCC_reduction;
constexpr auto ACCC_self = Clause::ACCC_self;
constexpr auto ACCC_seq = Clause::ACCC_seq;
constexpr auto ACCC_shortloop = Clause::ACCC_shortloop;
constexpr auto ACCC_tile = Clause::ACCC_tile;
constexpr auto ACCC_unknown = Clause::ACCC_unknown;
constexpr auto ACCC_use_device = Clause::ACCC_use_device;
constexpr auto ACCC_vector = Clause::ACCC_vector;
constexpr auto ACCC_vector_length = Clause::ACCC_vector_length;
constexpr auto ACCC_wait = Clause::ACCC_wait;
constexpr auto ACCC_worker = Clause::ACCC_worker;
constexpr auto ACCC_write = Clause::ACCC_write;

enum class DefaultValue {
  ACC_Default_present=0,
  ACC_Default_none=1,
};

constexpr auto ACC_Default_present = DefaultValue::ACC_Default_present;
constexpr auto ACC_Default_none = DefaultValue::ACC_Default_none;

// Enumeration helper functions
LLVM_ABI std::pair<Directive, directive::VersionRange> getOpenACCDirectiveKindAndVersions(StringRef Str);
inline Directive getOpenACCDirectiveKind(StringRef Str) {
  return getOpenACCDirectiveKindAndVersions(Str).first;
}

LLVM_ABI StringRef getOpenACCDirectiveName(Directive D, unsigned Ver = 0);

LLVM_ABI std::pair<Clause, directive::VersionRange> getOpenACCClauseKindAndVersions(StringRef Str);

inline Clause getOpenACCClauseKind(StringRef Str) {
  return getOpenACCClauseKindAndVersions(Str).first;
}

LLVM_ABI StringRef getOpenACCClauseName(Clause C, unsigned Ver = 0);

/// Return true if \p C is a valid clause for \p D in version \p Version.
LLVM_ABI bool isAllowedClauseForDirective(Directive D, Clause C, unsigned Version);

constexpr std::size_t getMaxLeafCount() { return 2; }
LLVM_ABI Association getDirectiveAssociation(Directive D);
LLVM_ABI Category getDirectiveCategory(Directive D);
LLVM_ABI SourceLanguage getDirectiveLanguages(Directive D);
LLVM_ABI DefaultValue getDefaultValue(StringRef Str);
LLVM_ABI StringRef getOpenACCDefaultValueName(DefaultValue x);

} // namespace acc

template <> struct enum_iteration_traits<acc::Association> {
  static constexpr bool is_iterable = true;
};

template <> struct enum_iteration_traits<acc::Category> {
  static constexpr bool is_iterable = true;
};

template <> struct enum_iteration_traits<acc::Directive> {
  static constexpr bool is_iterable = true;
};

template <> struct enum_iteration_traits<acc::Clause> {
  static constexpr bool is_iterable = true;
};

} // namespace llvm

#endif // LLVM_OpenACC_INC

