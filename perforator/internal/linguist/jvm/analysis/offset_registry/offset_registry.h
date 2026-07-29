#pragma once

#include <util/system/types.h>

#include <string_view>

namespace NPerforator::NLinguist::NJvm {

struct TStructEntryLayout {
    size_t Stride;
    size_t StructNameOffset;
    size_t FieldNameOffset;
    size_t TypeNameOffset;
    size_t IsStaticOffset;
    size_t OffsetOffset;
    size_t AddressOffset;

    const void* Inc(const void* ptr) const {
        return reinterpret_cast<const char*>(ptr) + Stride;
    }

    char const* StructName(const void* ptr) const {
        return *reinterpret_cast<char const* const*>(reinterpret_cast<const char*>(ptr) + StructNameOffset);
    }

    char const* FieldName(const void* ptr) const {
        return *reinterpret_cast<char const* const*>(reinterpret_cast<const char*>(ptr) + FieldNameOffset);
    }

    char const* TypeName(const void* ptr) const {
        return *reinterpret_cast<char const* const*>(reinterpret_cast<const char*>(ptr) + TypeNameOffset);
    }

    ui64 IsStatic(const void* ptr) const {
        return *reinterpret_cast<const ui64*>(reinterpret_cast<const char*>(ptr) + IsStaticOffset);
    }

    ui64 Offset(const void* ptr) const {
        return *reinterpret_cast<const ui64*>(reinterpret_cast<const char*>(ptr) + OffsetOffset);
    }

    void* Address(const void* ptr) const {
        return *reinterpret_cast<void* const*>(reinterpret_cast<const char*>(ptr) + AddressOffset);
    }
};

struct TTypeEntryLayout {
    size_t Stride;
    size_t StructNameOffset;
    size_t SuperNameOffset;
    size_t IsOopOffset;
    size_t IsIntegerOffset;
    size_t IsUnsignedOffset;
    size_t SizeOffset;

    char const* StructName(const void* ptr) const {
        return *reinterpret_cast<char const* const*>(reinterpret_cast<const char*>(ptr) + StructNameOffset);
    }

    char const* SuperName(const void* ptr) const {
        return *reinterpret_cast<char const* const*>(reinterpret_cast<const char*>(ptr) + SuperNameOffset);
    }

    i32 IsOop(const void* ptr) const {
        return *reinterpret_cast<const i32*>(reinterpret_cast<const char*>(ptr) + IsOopOffset);
    }

    i32 IsInteger(const void* ptr) const {
        return *reinterpret_cast<const i32*>(reinterpret_cast<const char*>(ptr) + IsIntegerOffset);
    }

    i32 IsUnsigned(const void* ptr) const {
        return *reinterpret_cast<const i32*>(reinterpret_cast<const char*>(ptr) + IsUnsignedOffset);
    }

    ui64 Size(const void* ptr) const {
        return *reinterpret_cast<const ui64*>(reinterpret_cast<const char*>(ptr) + SizeOffset);
    }
};

struct TIntEntryLayout {
    size_t Stride;
    size_t NameOffset;
    size_t ValueOffset;

    char const* Name(const void* ptr) const {
        return *reinterpret_cast<char const* const*>(reinterpret_cast<const char*>(ptr) + NameOffset);
    }

    i32 Value(const void* ptr) const {
        return *reinterpret_cast<const i32*>(reinterpret_cast<const char*>(ptr) + ValueOffset);
    }
};

template<typename T>
class TSpan {
public:
    TSpan(const void* data, size_t count, T layout)
        : Data_(data)
        , Count_(count)
        , Layout_(layout)
    {}

    const T& Layout() const {
        return Layout_;
    }

    class TSentinel{};

    class TIter {
        friend class TSpan;
    public:
        const void* operator*() const {
            return Elem_;
        }
        TIter& operator++() {
            Elem_ += Layout_.Stride;
            RemItems_--;
            return *this;
        }

        bool operator!=(const TSentinel&) const {
            return RemItems_ > 0;
        };
    private:
        TIter(T layout, const void* elem, size_t remItems)
            : Elem_(reinterpret_cast<const char*>(elem))
            , RemItems_(remItems)
            , Layout_(layout)
        {}
    private:
        const char* Elem_;
        size_t RemItems_;
        T Layout_;
    };

    TIter begin() const {
        return TIter{Layout_, Data_, Count_};
    }

    TSentinel end() const {
        return {};
    }
private:
    const void* Data_;
    size_t Count_;
    T Layout_;
};

class TJvmMetadata {
private:
    TSpan<TStructEntryLayout> Structs_;
    TSpan<TTypeEntryLayout> Types_;
    TSpan<TIntEntryLayout> Ints_;

public:
    TJvmMetadata(
        TSpan<TStructEntryLayout> structs,
        TSpan<TTypeEntryLayout> types,
        TSpan<TIntEntryLayout> ints
    );

private:
    const void* FindField(std::string_view typeName, std::string_view fieldName) const;

public:
    uintptr_t FindStaticFieldAddress(std::string_view typeName, std::string_view fieldName) const;

    size_t FindFieldOffset(std::string_view typeName, std::string_view fieldName) const;

    size_t FindTypeSize(std::string_view typeName) const;

    i32 FindIntValue(std::string_view intName) const;
};

}
