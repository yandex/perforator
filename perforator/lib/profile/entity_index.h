#pragma once

#include <util/generic/yexception.h>

#include <algorithm>
#include <compare>


namespace NPerforator::NProfile {

////////////////////////////////////////////////////////////////////////////////

template <typename Tag = decltype([]{})>
class TStrongIndex {
    static constexpr inline i32 InvalidIndexValue = Min<i32>();

public:
    // Do not use default constructor.
    // Use static constructor-like factory functions defined below.
    TStrongIndex() = delete;

    static TStrongIndex Invalid() {
        return TStrongIndex{InvalidIndexValue};
    }

    static TStrongIndex Zero() {
        return FromInternalIndex(0);
    }

    static TStrongIndex FromInternalIndex(TExplicitType<i32> index) {
        Y_ENSURE(index >= 0);
        return TStrongIndex{index};
    }

    static TStrongIndex FromInternalIndex(TExplicitType<ui32> index) {
        Y_ENSURE(index < static_cast<ui32>(Max<i32>()));
        return FromInternalIndex(static_cast<i32>(index));
    }

public:
    std::strong_ordering operator<=>(const TStrongIndex& rhs) const noexcept = default;

    i32 GetInternalIndex() const {
        Y_ASSERT(IsValid());
        return Value_;
    }

    i32 operator*() const {
        Y_ASSERT(IsValid());
        return Value_;
    }

    bool IsValid() const {
        return Value_ >= 0;
    }

    template <typename H>
    friend H AbslHashValue(H hash, const TStrongIndex& index) {
        return H::combine(std::move(hash), index.Value_);
    }

private:
    explicit TStrongIndex(i32 value)
        : Value_{value}
    {
        Y_ASSERT(Value_ >= 0 || Value_ == InvalidIndexValue);
    }

private:
    i32 Value_ = 0;
};

#define Y_DEFINE_STRONG_INDEX_TAG(Name) \
    struct Name ## Tag {}; \
    \
    using Name = ::NPerforator::NProfile::TStrongIndex<Name ## Tag>;

////////////////////////////////////////////////////////////////////////////////

template <typename T>
inline constexpr bool IsStrongIndex = false;

template <typename Tag>
inline constexpr bool IsStrongIndex<TStrongIndex<Tag>> = true;

template <typename T>
concept CStrongIndex = IsStrongIndex<T>;

////////////////////////////////////////////////////////////////////////////////

Y_DEFINE_STRONG_INDEX_TAG(TStringId);
Y_DEFINE_STRONG_INDEX_TAG(TCommentId);
Y_DEFINE_STRONG_INDEX_TAG(TValueTypeId);
Y_DEFINE_STRONG_INDEX_TAG(TSampleId);
Y_DEFINE_STRONG_INDEX_TAG(TSampleKeyId);
Y_DEFINE_STRONG_INDEX_TAG(TStackId);
Y_DEFINE_STRONG_INDEX_TAG(TBinaryId);
Y_DEFINE_STRONG_INDEX_TAG(TStackFrameId);
Y_DEFINE_STRONG_INDEX_TAG(TInlineChainId);
Y_DEFINE_STRONG_INDEX_TAG(TSourceLineId);
Y_DEFINE_STRONG_INDEX_TAG(TFunctionId);
Y_DEFINE_STRONG_INDEX_TAG(TThreadId);
Y_DEFINE_STRONG_INDEX_TAG(TLabelId);

////////////////////////////////////////////////////////////////////////////////

} // namespace NPerforator::NProfile
