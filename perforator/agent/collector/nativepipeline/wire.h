#pragma once

#include <perforator/agent/collector/progs/unwinder/contract/wire_types.h>

#include <cstddef>
#include <type_traits>

namespace NPerforator::NAgent::NNativePipeline::NWire {

inline constexpr std::size_t MaxPackedDataSize = PACKED_SAMPLE_MAX_DATA;

static_assert(std::is_trivially_copyable_v<record_sample_header>);
static_assert(sizeof(perf_event_attr_subset) == 16);
static_assert(sizeof(perf_event_subset) == 24);
static_assert(sizeof(sample_config) == 24);
static_assert(sizeof(section_desc) == 4);
static_assert(sizeof(record_sample_header) == 152);
static_assert(offsetof(record_sample_header, sample_type) == 4);
static_assert(offsetof(record_sample_header, sample_config) == 8);
static_assert(offsetof(record_sample_header, runtime) == 36);
static_assert(offsetof(record_sample_header, collection_time) == 40);
static_assert(offsetof(record_sample_header, pid) == 80);
static_assert(offsetof(record_sample_header, parent_cgroup) == 96);
static_assert(offsetof(record_sample_header, kern_stack) == 128);
static_assert(offsetof(record_sample_header, language_sections) == 148);
static_assert(sizeof(branch_record) == 24);
static_assert(sizeof(language_section_header) == 8);
static_assert(offsetof(language_section_header, language) == 2);
static_assert(offsetof(language_section_header, element_size) == 4);
static_assert(sizeof(thread_local_variable_collect_result) == 152);
static_assert(offsetof(thread_local_variable_collect_result, type) == 8);
static_assert(offsetof(thread_local_variable_collect_result, value) == 16);

} // namespace NPerforator::NAgent::NNativePipeline::NWire
