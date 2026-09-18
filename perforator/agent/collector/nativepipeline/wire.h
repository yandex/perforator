#pragma once

#include <cstddef>
#include <cstdint>
#include <type_traits>

namespace NPerforator::NAgent::NNativePipeline::NWire {

#define PERFORATOR_WIRE_U8 std::uint8_t
#define PERFORATOR_WIRE_U16 std::uint16_t
#define PERFORATOR_WIRE_U32 std::uint32_t
#define PERFORATOR_WIRE_U64 std::uint64_t
#define PERFORATOR_WIRE_I32 std::int32_t
#define PERFORATOR_WIRE_FIELD(bpf_name, host_name) host_name
#define PERFORATOR_WIRE_ENUM(bpf_name, host_name) host_name
#include <perforator/agent/collector/progs/unwinder/contract/wire_types.inc>
#undef PERFORATOR_WIRE_U8
#undef PERFORATOR_WIRE_U16
#undef PERFORATOR_WIRE_U32
#undef PERFORATOR_WIRE_U64
#undef PERFORATOR_WIRE_I32
#undef PERFORATOR_WIRE_FIELD
#undef PERFORATOR_WIRE_ENUM

using ERecordTag = record_tag;
using ESampleType = sample_type;
using ELanguage = language_id;
using ELanguagePayloadKind = language_payload_kind;
using ETlsVariableType = tls_variable_type;
using TPerfEventAttrSubset = perf_event_attr_subset;
using TPerfEventSubset = perf_event_subset;
using TSampleConfig = sample_config;
using TSectionDesc = section_desc;
using TRecordSampleHeader = record_sample_header;
using TBranchRecord = branch_record;
using TJvmLanguageEntry = jvm_lang_entry;
using TLanguageSectionHeader = language_section_header;
using TTlsCollectResult = thread_local_variable_collect_result;

inline constexpr std::size_t MaxPackedDataSize = PACKED_SAMPLE_MAX_DATA;
inline constexpr std::size_t ParentCgroupMaxLevels = PARENT_CGROUP_MAX_LEVELS;
inline constexpr std::uint64_t EndOfCgroupList = END_OF_CGROUP_LIST;

static_assert(std::is_trivially_copyable_v<TRecordSampleHeader>);
static_assert(sizeof(TPerfEventAttrSubset) == 16);
static_assert(sizeof(TPerfEventSubset) == 24);
static_assert(sizeof(TSampleConfig) == 24);
static_assert(sizeof(TSectionDesc) == 4);
static_assert(sizeof(TRecordSampleHeader) == 152);
static_assert(offsetof(TRecordSampleHeader, SampleType) == 4);
static_assert(offsetof(TRecordSampleHeader, SampleConfig) == 8);
static_assert(offsetof(TRecordSampleHeader, Runtime) == 36);
static_assert(offsetof(TRecordSampleHeader, CollectionTime) == 40);
static_assert(offsetof(TRecordSampleHeader, Pid) == 80);
static_assert(offsetof(TRecordSampleHeader, ParentCgroup) == 96);
static_assert(offsetof(TRecordSampleHeader, KernStack) == 128);
static_assert(offsetof(TRecordSampleHeader, LanguageSections) == 148);
static_assert(sizeof(TBranchRecord) == 24);
static_assert(sizeof(TJvmLanguageEntry) == 16);
static_assert(sizeof(TLanguageSectionHeader) == 8);
static_assert(offsetof(TLanguageSectionHeader, PayloadKind) == 3);
static_assert(offsetof(TLanguageSectionHeader, ElementSize) == 4);
static_assert(sizeof(TTlsCollectResult) == 152);
static_assert(offsetof(TTlsCollectResult, Type) == 8);
static_assert(offsetof(TTlsCollectResult, Value) == 16);

} // namespace NPerforator::NAgent::NNativePipeline::NWire
