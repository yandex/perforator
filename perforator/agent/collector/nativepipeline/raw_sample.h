#pragma once

#include "wire.h"

#include <util/generic/strbuf.h>

#include <cstddef>
#include <cstring>
#include <array>

namespace NPerforator::NAgent::NNativePipeline {

////////////////////////////////////////////////////////////////////////////////

class TRawSection {
public:
    TRawSection() = default;
    TRawSection(const char* data, std::size_t size)
        : Data_{data}
        , Size_{size}
    {
    }

    bool Empty() const noexcept {
        return Size_ == 0;
    }

    TStringBuf Bytes() const noexcept {
        return TStringBuf{Data_, Size_};
    }

    template <typename T>
    std::size_t Count() const noexcept {
        return Size_ / sizeof(T);
    }

    // Perf-buffer records are normally aligned, but memcpy keeps the parser
    // correct for arbitrary byte slices used by tests and fuzzing.
    template <typename T>
    T Read(std::size_t index) const {
        T value;
        std::memcpy(&value, Data_ + index * sizeof(T), sizeof(T));
        return value;
    }

private:
    const char* Data_ = nullptr;
    std::size_t Size_ = 0;
};

class TRawLanguageSection {
public:
    NWire::ELanguage Language() const noexcept {
        return Language_;
    }

    NWire::ELanguagePayloadKind PayloadKind() const noexcept {
        return PayloadKind_;
    }

    std::size_t ElementSize() const noexcept {
        return ElementSize_;
    }

    std::size_t Count() const noexcept {
        return ElementSize_ == 0 ? 0 : Data_.Bytes().size() / ElementSize_;
    }

    TStringBuf Element(std::size_t index) const noexcept {
        const auto bytes = Data_.Bytes();
        if (ElementSize_ == 0 || index >= Count()) {
            return {};
        }
        return {bytes.data() + index * ElementSize_, ElementSize_};
    }

private:
    friend class TRawSample;

    NWire::ELanguage Language_{};
    NWire::ELanguagePayloadKind PayloadKind_{};
    std::size_t ElementSize_ = 0;
    TRawSection Data_;
};

class TRawSample {
public:
    // The parsed sample borrows record bytes. The caller must keep record
    // alive until all sections have been consumed.
    static TRawSample Parse(TStringBuf record);

    const NWire::TRecordSampleHeader& Header() const noexcept {
        return Header_;
    }

    TRawSection KernelStack() const noexcept;
    TRawSection UserStack() const noexcept;
    TRawSection Lbr() const noexcept;
    TRawSection Tls() const noexcept;
    TRawSection Cgroups() const noexcept;
    TRawSection LanguageSections() const noexcept;
    const TRawLanguageSection* LanguageSection(NWire::ELanguage language) const noexcept;
    std::size_t LanguageSectionCount() const noexcept {
        return LanguageCount_;
    }
    const TRawLanguageSection& LanguageSectionAt(std::size_t index) const noexcept {
        return Languages_[index];
    }

private:
    TRawSection GetSection(const NWire::TSectionDesc& desc) const noexcept;

private:
    NWire::TRecordSampleHeader Header_{};
    const char* Data_ = nullptr;
    static constexpr std::size_t MaxLanguageSections = PERFORATOR_LANGUAGE_COUNT;
    std::array<TRawLanguageSection, MaxLanguageSections> Languages_{};
    std::size_t LanguageCount_ = 0;
};

////////////////////////////////////////////////////////////////////////////////

} // namespace NPerforator::NAgent::NNativePipeline
