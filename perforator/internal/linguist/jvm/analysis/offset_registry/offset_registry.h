#pragma once

#include <util/system/types.h>

#include <span>
#include <string_view>

namespace NPerforator::NLinguist::NJvm {

struct THotSpotStructEntry {
    const char* StructName;
    const char* FieldName;
    const char* TypeName;
    ui64 IsStatic;
    ui64 Offset;
    void* Address;
};

struct THotSpotTypeEntry {
    const char* StructName;
    const char* SuperName;
    i32 IsOop;
    i32 IsInteger;
    i32 IsUnsigned;
    ui64 Size;
};

struct IntLayout {
    size_t Stride;
    size_t NameOffset;
    size_t ValueOffset;

    const void* Inc(const void* ptr) const {
        return reinterpret_cast<const char*>(ptr) + Stride;
    }

    char const* Name(const void* ptr) const {
        return *reinterpret_cast<char const* const*>(ptr) + NameOffset;
    }

    i32 Value(const void* ptr) const {
        return *reinterpret_cast<const i32*>(reinterpret_cast<const char*>(ptr) + ValueOffset);
    }
};

class TJvmMetadata {
private:
    std::span<const THotSpotStructEntry> Structs_;
    std::span<const THotSpotTypeEntry> Types_;
    const void* IntsData_;
    size_t IntsCount_;
    IntLayout IntLayout_;

public:
    TJvmMetadata(
        std::span<const THotSpotStructEntry> structs,
        std::span<const THotSpotTypeEntry> types,
        const void* intsData,
        size_t intsCount,
        IntLayout intLayout
    );

private:
    const THotSpotStructEntry* FindField(std::string_view typeName, std::string_view fieldName) const;

public:
    uintptr_t FindStaticFieldAddress(std::string_view typeName, std::string_view fieldName) const;

    size_t FindFieldOffset(std::string_view typeName, std::string_view fieldName) const;

    size_t FindTypeSize(std::string_view typeName) const;

    i32 FindIntValue(std::string_view intName) const;
};

}
