#pragma once

#include "region_data_provider.h"

#include <util/generic/array_ref.h>
#include <util/generic/strbuf.h>

#include <google/protobuf/wire_format_lite.h>

namespace NInPlaceProto {

// ======================= Public API =======================

enum class EPackedType {
    Int32,
    Int64,
    UInt32,
    UInt64,
    SInt32,
    SInt64,
    Bool,
    Fixed32,
    Fixed64,
    SFixed32,
    SFixed64,
    Float,
    Double,
};

template <EPackedType PackedType>
class TPackedRepeatedView;

// ====================== Private impl ======================

namespace NPackedRepeatedDetail {

    using WireFormatLite = ::google::protobuf::internal::WireFormatLite;

    enum class EPackedEncoding {
        Varint,
        ZigZag,
        Fixed32,
        Fixed64,
    };

    template <typename T, EPackedEncoding Encoding>
    struct TPackedTraits;

    template <>
    struct TPackedTraits<float, EPackedEncoding::Fixed32> {
        static constexpr size_t ElementSize = 4;
        static constexpr bool IsFixed = true;
        static float Decode(const ui8* ptr) noexcept {
            return WireFormatLite::DecodeFloat(LittleToHost(*reinterpret_cast<const ui32*>(ptr)));
        }
    };
    template <>
    struct TPackedTraits<ui32, EPackedEncoding::Fixed32> {
        static constexpr size_t ElementSize = 4;
        static constexpr bool IsFixed = true;
        static ui32 Decode(const ui8* ptr) noexcept {
            return LittleToHost(*reinterpret_cast<const ui32*>(ptr));
        }
    };
    template <>
    struct TPackedTraits<i32, EPackedEncoding::Fixed32> {
        static constexpr size_t ElementSize = 4;
        static constexpr bool IsFixed = true;
        static i32 Decode(const ui8* ptr) noexcept {
            return static_cast<i32>(LittleToHost(*reinterpret_cast<const ui32*>(ptr)));
        }
    };

    template <>
    struct TPackedTraits<double, EPackedEncoding::Fixed64> {
        static constexpr size_t ElementSize = 8;
        static constexpr bool IsFixed = true;
        static double Decode(const ui8* ptr) noexcept {
            return WireFormatLite::DecodeDouble(LittleToHost(*reinterpret_cast<const ui64*>(ptr)));
        }
    };
    template <>
    struct TPackedTraits<ui64, EPackedEncoding::Fixed64> {
        static constexpr size_t ElementSize = 8;
        static constexpr bool IsFixed = true;
        static ui64 Decode(const ui8* ptr) noexcept {
            return LittleToHost(*reinterpret_cast<const ui64*>(ptr));
        }
    };
    template <>
    struct TPackedTraits<i64, EPackedEncoding::Fixed64> {
        static constexpr size_t ElementSize = 8;
        static constexpr bool IsFixed = true;
        static i64 Decode(const ui8* ptr) noexcept {
            return static_cast<i64>(LittleToHost(*reinterpret_cast<const ui64*>(ptr)));
        }
    };

    template <typename T>
    struct TPackedTraits<T, EPackedEncoding::Varint> {
        static constexpr bool IsFixed = false;
        static T DecodeVarint(ui64 raw) noexcept {
            return static_cast<T>(raw);
        }
    };

    template <>
    struct TPackedTraits<i32, EPackedEncoding::ZigZag> {
        static constexpr bool IsFixed = false;
        static i32 DecodeVarint(ui64 raw) noexcept {
            return static_cast<i32>(raw >> 1) ^ -static_cast<i32>(raw & 1);
        }
    };
    template <>
    struct TPackedTraits<i64, EPackedEncoding::ZigZag> {
        static constexpr bool IsFixed = false;
        static i64 DecodeVarint(ui64 raw) noexcept {
            return static_cast<i64>(raw >> 1) ^ -static_cast<i64>(raw & 1);
        }
    };

    template <EPackedType>
    struct TPackedTypeTraits;

    template <> struct TPackedTypeTraits<EPackedType::Float> : TPackedTraits<float, EPackedEncoding::Fixed32> {
        using TValue = float;
    };
    template <> struct TPackedTypeTraits<EPackedType::Double> : TPackedTraits<double, EPackedEncoding::Fixed64> {
        using TValue = double;
    };
    template <> struct TPackedTypeTraits<EPackedType::Fixed32> : TPackedTraits<ui32, EPackedEncoding::Fixed32> {
        using TValue = ui32;
    };
    template <> struct TPackedTypeTraits<EPackedType::SFixed32> : TPackedTraits<i32, EPackedEncoding::Fixed32> {
        using TValue = i32;
    };
    template <> struct TPackedTypeTraits<EPackedType::Fixed64> : TPackedTraits<ui64, EPackedEncoding::Fixed64> {
        using TValue = ui64;
    };
    template <> struct TPackedTypeTraits<EPackedType::SFixed64> : TPackedTraits<i64, EPackedEncoding::Fixed64> {
        using TValue = i64;
    };
    template <> struct TPackedTypeTraits<EPackedType::Int32> : TPackedTraits<i32, EPackedEncoding::Varint> {
        using TValue = i32;
    };
    template <> struct TPackedTypeTraits<EPackedType::Int64> : TPackedTraits<i64, EPackedEncoding::Varint> {
        using TValue = i64;
    };
    template <> struct TPackedTypeTraits<EPackedType::UInt32> : TPackedTraits<ui32, EPackedEncoding::Varint> {
        using TValue = ui32;
    };
    template <> struct TPackedTypeTraits<EPackedType::UInt64> : TPackedTraits<ui64, EPackedEncoding::Varint> {
        using TValue = ui64;
    };
    template <> struct TPackedTypeTraits<EPackedType::SInt32> : TPackedTraits<i32, EPackedEncoding::ZigZag> {
        using TValue = i32;
    };
    template <> struct TPackedTypeTraits<EPackedType::SInt64> : TPackedTraits<i64, EPackedEncoding::ZigZag> {
        using TValue = i64;
    };
    template <> struct TPackedTypeTraits<EPackedType::Bool> : TPackedTraits<bool, EPackedEncoding::Varint> {
        using TValue = bool;
    };

    inline size_t CountVarintElements(const char* data, size_t size) noexcept {
        const ui8* p = reinterpret_cast<const ui8*>(data);
        const ui8* end = p + size;
        size_t count = 0;
        for (; p < end; ++p) {
            count += static_cast<size_t>((*p & 0x80) == 0);
        }
        return count;
    }

} // namespace NPackedRepeatedDetail

// =================== TPackedRepeatedView ==================

/* Packed repeated field view — specify the proto field type explicitly:
 *
 *   parser.GetPackedRepeatedView<EPackedType::UInt32>()     // repeated uint32   [packed=true]
 *   parser.GetPackedRepeatedView<EPackedType::Float>()      // repeated float    [packed=true]
 *   parser.GetPackedRepeatedView<EPackedType::SInt32>()     // repeated sint32   [packed=true]
 *
 *   for (float f : parser.GetPackedRepeatedView<EPackedType::Float>()) { ... }
 *
 * Fixed types (Float, Double, Fixed32/64, SFixed32/64):
 *   size() — O(1), operator[] — O(1), begin()/end() — random-access.
 *
 * Varint types (Int32/64, UInt32/64, SInt32/64, Bool):
 *   begin()/end() — forward-only, range-based for. No operator[].
 *
 * The view is always tied to a TRegionDataProvider (typically the parser).
 * Varint corruption detected during iteration is reported via SetCorrupted().
 * The view must not outlive the provider.
 */
template <EPackedType PackedType>
class TPackedRepeatedView {
    using TTraits = NPackedRepeatedDetail::TPackedTypeTraits<PackedType>;
    using T = typename TTraits::TValue;
    static constexpr bool IsFixed = TTraits::IsFixed;

public:
    explicit TPackedRepeatedView(TStringBuf data, TRegionDataProvider& provider) noexcept
        : Data_(data)
        , ExternalProvider_(provider)
    {}

    explicit TPackedRepeatedView(TArrayRef<const char> data, TRegionDataProvider& provider) noexcept
        : Data_(data.data(), data.size())
        , ExternalProvider_(provider)
    {}

    bool empty() const noexcept {
        if constexpr (IsFixed) {
            return Data_.size() < TTraits::ElementSize;
        } else {
            return Data_.empty();
        }
    }

    bool IsCorrupted() const noexcept {
        return ExternalProvider_.IsCorrupted();
    }

    const TStringBuf& GetRawBuf() const noexcept {
        return Data_;
    }

    template <bool Fixed = IsFixed, typename = std::enable_if_t<Fixed>>
    size_t size() const noexcept {
        return Data_.size() / TTraits::ElementSize;
    }

    template <bool Fixed = IsFixed, typename = std::enable_if_t<Fixed>>
    T operator[](size_t index) const noexcept {
        const ui8* ptr = reinterpret_cast<const ui8*>(Data_.data()) + index * TTraits::ElementSize;
        return TTraits::Decode(ptr);
    }

    struct TIteratorEnd {};

    class TFixedIterator {
    public:
        using value_type = T;
        using difference_type = ptrdiff_t;

        TFixedIterator(const ui8* ptr, const ui8* end) noexcept
            : Provider_(ptr, end)
        {
            HasValue_ = Provider_.CanRead(TTraits::ElementSize);
        }

        bool HasValue() const noexcept {
            return HasValue_;
        }

        T operator*() const noexcept {
            return TTraits::Decode(Provider_.GetCurrentPos());
        }

        TFixedIterator& operator++() noexcept {
            Provider_.Skip(static_cast<ui32>(TTraits::ElementSize));
            HasValue_ = Provider_.CanRead(TTraits::ElementSize);
            return *this;
        }

        TFixedIterator& operator+=(difference_type n) noexcept {
            Provider_.Skip(static_cast<ui32>(n * TTraits::ElementSize));
            HasValue_ = Provider_.CanRead(TTraits::ElementSize);
            return *this;
        }

        bool operator==(TIteratorEnd) const noexcept {
            return !HasValue_;
        }

        bool operator!=(TIteratorEnd) const noexcept {
            return HasValue_;
        }

    private:
        TRegionDataProvider Provider_;
        bool HasValue_ = false;
    };

    class TVarintIterator {
    public:
        using value_type = T;
        using difference_type = ptrdiff_t;

        TVarintIterator(const ui8* ptr, const ui8* end, TRegionDataProvider& externalProvider) noexcept
            : Provider_(ptr, end)
            , ExternalProvider_(externalProvider)
        {
            FetchNext();
        }

        bool HasValue() const noexcept {
            return HasValue_;
        }

        const T& operator*() const noexcept {
            return CachedValue_;
        }

        TVarintIterator& operator++() noexcept {
            FetchNext();
            return *this;
        }

        bool operator==(TIteratorEnd) const noexcept {
            return !HasValue_;
        }

        bool operator!=(TIteratorEnd) const noexcept {
            return HasValue_;
        }

    private:
        void FetchNext() noexcept {
            if (!Provider_.NotEmpty()) {
                HasValue_ = false;
                return;
            }
            ui64 raw = Provider_.ReadVarint64();
            ExternalProvider_.UpdateCorrupted(Provider_.IsCorrupted());
            HasValue_ = !Provider_.IsCorrupted();
            CachedValue_ = TTraits::DecodeVarint(raw);
        }

        TRegionDataProvider Provider_;
        TRegionDataProvider& ExternalProvider_;
        bool HasValue_ = false;
        T CachedValue_{};
    };

    using TIterator = std::conditional_t<IsFixed, TFixedIterator, TVarintIterator>;

    TIterator begin() const noexcept {
        const ui8* start = reinterpret_cast<const ui8*>(Data_.data());
        const ui8* finish = start + Data_.size();
        if constexpr (IsFixed) {
            return TFixedIterator(start, finish);
        } else {
            return TVarintIterator(start, finish, ExternalProvider_);
        }
    }

    TIteratorEnd end() const noexcept {
        return {};
    }

private:
    TStringBuf Data_;
    TRegionDataProvider& ExternalProvider_;
};

} // namespace NInPlaceProto
