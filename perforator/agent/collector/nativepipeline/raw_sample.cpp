#include "raw_sample.h"

#include <util/generic/yexception.h>

#include <array>
#include <cstdint>
#include <cstring>

namespace NPerforator::NAgent::NNativePipeline {
namespace {

////////////////////////////////////////////////////////////////////////////////

using namespace NWire;

constexpr std::size_t PackedAlignment = sizeof(std::uint64_t);
constexpr std::size_t MaxPackedSize = sizeof(TRecordSampleHeader) + MaxPackedDataSize;
static_assert(MaxPackedSize % PackedAlignment == 0);
// PERF_SAMPLE_RAW includes the padding that aligns its u32 size prefix
// plus the payload to 8 bytes. Our packed payload is already 8-byte
// aligned, so perf adds four trailing bytes (not necessarily zero).
constexpr std::size_t PerfPaddingSize = PackedAlignment - sizeof(std::uint32_t);

struct TSectionSpec {
    const TSectionDesc* Desc;
    std::size_t ElementSize;
    TStringBuf Name;
};

} // anonymous namespace

TRawSample TRawSample::Parse(TStringBuf record) {
    using namespace NWire;

    if (record.size() < sizeof(TRecordSampleHeader)) {
        ythrow yexception() << "packed sample is too short: " << record.size();
    }
    if (record.size() > MaxPackedSize + PerfPaddingSize) {
        ythrow yexception() << "packed sample is too large: " << record.size();
    }

    // Validate the wire byte before it becomes a bool in the copied header.
    const unsigned kthread = static_cast<unsigned char>(record[offsetof(TRecordSampleHeader, kthread)]);
    if (kthread > 1) {
        ythrow yexception() << "invalid kthread flag " << kthread;
    }

    TRawSample result;
    std::memcpy(&result.Header_, record.data(), sizeof(result.Header_));
    result.Data_ = record.data() + sizeof(result.Header_);
    // Every packed section has an 8-byte-aligned size. Exclude transport
    // padding from bounds checks, for both fixed-size 5.4 records and
    // variable-size records, while still accepting unpadded payloads.
    const std::size_t dataSize = (record.size() - sizeof(result.Header_)) / PackedAlignment * PackedAlignment;

    if (result.Header_.tag != ERecordTag::RECORD_TAG_SAMPLE) {
        ythrow yexception() << "unexpected perf record tag "
                            << static_cast<unsigned>(result.Header_.tag);
    }
    if (result.Header_.sample_type <= ESampleType::SAMPLE_TYPE_UNDEFINED ||
        result.Header_.sample_type > ESampleType::SAMPLE_TYPE_UPROBE)
    {
        ythrow yexception() << "unknown sample type "
                            << static_cast<std::uint32_t>(result.Header_.sample_type);
    }

    const std::array<TSectionSpec, 6> sections{{
        {&result.Header_.kern_stack, sizeof(std::uint64_t), "kernel stack"},
        {&result.Header_.user_stack, sizeof(std::uint64_t), "user stack"},
        {&result.Header_.lbr, sizeof(TBranchRecord), "last branch records"},
        {&result.Header_.tls, sizeof(TTlsCollectResult), "TLS"},
        {&result.Header_.cgroups, sizeof(std::uint64_t), "cgroups"},
        {&result.Header_.language_sections, 1, "language sections"},
    }};

    std::size_t expectedOffset = 0;
    for (const auto& section : sections) {
        const std::size_t offset = section.Desc->offset;
        const std::size_t size = section.Desc->size;
        if (offset != expectedOffset) {
            ythrow yexception() << section.Name << " section is not contiguous: expected offset "
                                << expectedOffset << ", got " << offset;
        }
        if (offset % 8 != 0 || size % section.ElementSize != 0) {
            ythrow yexception() << "invalid " << section.Name << " section alignment/size";
        }
        if (offset > dataSize || size > dataSize - offset) {
            ythrow yexception() << section.Name << " section exceeds packed sample";
        }
        expectedOffset += size;
    }

    const TStringBuf languages = result.LanguageSections().Bytes();
    std::size_t position = 0;
    while (position != languages.size()) {
        if (languages.size() - position < sizeof(TLanguageSectionHeader)) {
            ythrow yexception() << "truncated language section header at " << position;
        }
        TLanguageSectionHeader header;
        std::memcpy(&header, languages.data() + position, sizeof(header));
        position += sizeof(header);
        if (static_cast<std::uint8_t>(header.language) >= PERFORATOR_LANGUAGE_COUNT) {
            ythrow yexception() << "unknown language id "
                                << static_cast<unsigned>(header.language);
        }
        if (result.LanguageCount_ == result.Languages_.size()) {
            ythrow yexception() << "too many language sections";
        }
        for (std::size_t i = 0; i < result.LanguageCount_; ++i) {
            if (result.Languages_[i].Language() == header.language) {
                ythrow yexception() << "duplicate language section "
                                    << static_cast<unsigned>(header.language);
            }
        }
        if (header.payload_kind != ELanguagePayloadKind::LANGUAGE_PAYLOAD_INTERPRETER_FRAMES &&
            header.payload_kind != ELanguagePayloadKind::LANGUAGE_PAYLOAD_NATIVE_ANNOTATIONS)
        {
            ythrow yexception() << "unknown language payload kind "
                                << static_cast<unsigned>(header.payload_kind);
        }
        const std::size_t elementSize = header.element_size;
        if (elementSize == 0 || elementSize % alignof(std::uint64_t) != 0 ||
            header.byte_size == 0 || header.byte_size % elementSize != 0)
        {
            ythrow yexception() << "invalid language section size " << header.byte_size;
        }
        if (header.payload_kind == ELanguagePayloadKind::LANGUAGE_PAYLOAD_NATIVE_ANNOTATIONS &&
            (header.language != ELanguage::LANGUAGE_JVM || elementSize != sizeof(TJvmLanguageEntry)))
        {
            ythrow yexception() << "unsupported native language annotation layout";
        }
        if (header.byte_size > languages.size() - position) {
            ythrow yexception() << "language section exceeds packed sample";
        }
        auto& section = result.Languages_[result.LanguageCount_++];
        section.Language_ = header.language;
        section.PayloadKind_ = header.payload_kind;
        section.ElementSize_ = elementSize;
        section.Data_ = {languages.data() + position, header.byte_size};
        position += header.byte_size;
    }
    return result;
}

TRawSection TRawSample::GetSection(const NWire::TSectionDesc& desc) const noexcept {
    return {Data_ + desc.offset, desc.size};
}

TRawSection TRawSample::KernelStack() const noexcept {
    return GetSection(Header_.kern_stack);
}

TRawSection TRawSample::UserStack() const noexcept {
    return GetSection(Header_.user_stack);
}

TRawSection TRawSample::Lbr() const noexcept {
    return GetSection(Header_.lbr);
}

TRawSection TRawSample::Tls() const noexcept {
    return GetSection(Header_.tls);
}

TRawSection TRawSample::Cgroups() const noexcept {
    return GetSection(Header_.cgroups);
}

TRawSection TRawSample::LanguageSections() const noexcept {
    return GetSection(Header_.language_sections);
}

const TRawLanguageSection* TRawSample::LanguageSection(NWire::ELanguage language) const noexcept {
    for (std::size_t i = 0; i < LanguageCount_; ++i) {
        if (Languages_[i].Language() == language) {
            return &Languages_[i];
        }
    }
    return nullptr;
}

////////////////////////////////////////////////////////////////////////////////

} // namespace NPerforator::NAgent::NNativePipeline
