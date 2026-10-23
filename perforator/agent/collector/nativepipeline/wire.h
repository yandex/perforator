#pragma once

#include <perforator/agent/collector/progs/unwinder/contract/wire_types.h>

#include <cstddef>
#include <type_traits>

namespace NPerforator::NAgent::NNativePipeline::NWire {

using ERecordTag = record_tag;
using ESampleType = sample_type;
using ELanguage = language_id;
using ELanguagePayloadKind = language_payload_kind;
using ETlsVariableType = tls_variable_type;
using TSectionDesc = section_desc;
using TRecordSampleHeader = record_sample_header;
using TBranchRecord = branch_record;
using TJvmLanguageEntry = jvm_lang_entry;
using TLanguageSectionHeader = language_section_header;
using TTlsCollectResult = thread_local_variable_collect_result;

inline constexpr std::size_t MaxPackedDataSize = PACKED_SAMPLE_MAX_DATA;

static_assert(std::is_trivially_copyable_v<TRecordSampleHeader>);
static_assert(sizeof(perf_event_attr_subset) == 16);
static_assert(sizeof(perf_event_subset) == 24);
static_assert(sizeof(sample_config) == 24);
static_assert(sizeof(TSectionDesc) == 4);
static_assert(sizeof(TRecordSampleHeader) == 152);
static_assert(offsetof(TRecordSampleHeader, sample_type) == 4);
static_assert(offsetof(TRecordSampleHeader, sample_config) == 8);
static_assert(offsetof(TRecordSampleHeader, runtime) == 36);
static_assert(offsetof(TRecordSampleHeader, collection_time) == 40);
static_assert(offsetof(TRecordSampleHeader, pid) == 80);
static_assert(offsetof(TRecordSampleHeader, parent_cgroup) == 96);
static_assert(offsetof(TRecordSampleHeader, kern_stack) == 128);
static_assert(offsetof(TRecordSampleHeader, language_sections) == 148);
static_assert(sizeof(TBranchRecord) == 24);
static_assert(sizeof(TJvmLanguageEntry) == 16);
static_assert(sizeof(TLanguageSectionHeader) == 8);
static_assert(offsetof(TLanguageSectionHeader, payload_kind) == 3);
static_assert(offsetof(TLanguageSectionHeader, element_size) == 4);
static_assert(sizeof(TTlsCollectResult) == 152);
static_assert(offsetof(TTlsCollectResult, type) == 8);
static_assert(offsetof(TTlsCollectResult, value) == 16);

} // namespace NPerforator::NAgent::NNativePipeline::NWire
