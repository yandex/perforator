#include "offset_registry.h"

#include <util/stream/output.h>
#include <util/generic/yexception.h>


namespace NPerforator::NLinguist::NJvm {

TJvmMetadata::TJvmMetadata(
    TSpan<TStructEntryLayout> structs,
    TSpan<TTypeEntryLayout> types,
    TSpan<TIntEntryLayout> ints
)
    : Structs_(structs)
    , Types_(types)
    , Ints_(ints)
{
}

const void* TJvmMetadata::FindField(std::string_view typeName, std::string_view fieldName) const {
    for (const void* s : Structs_) {
        const char* sTn = Structs_.Layout().StructName(s);
        const char* sFn = Structs_.Layout().FieldName(s);
        if (sTn == nullptr || sFn == nullptr) {
            continue;
        }
        if (typeName == sTn && fieldName == sFn) {
            return s;
        }
    }
    return nullptr;
}

uintptr_t TJvmMetadata::FindStaticFieldAddress(std::string_view typeName, std::string_view fieldName) const {
    const void* s = FindField(typeName, fieldName);
    if (s == nullptr) {
        throw yexception() << "Static field " << fieldName << " not found in type " << typeName;
    }
    Y_THROW_UNLESS(Structs_.Layout().IsStatic(s));
    void* addr = Structs_.Layout().Address(s);
    Y_THROW_UNLESS(addr != nullptr);
    return reinterpret_cast<uintptr_t>(addr);
}

size_t TJvmMetadata::FindFieldOffset(std::string_view typeName, std::string_view fieldName) const {
    const void* s = FindField(typeName, fieldName);
    if (s == nullptr) {
        throw yexception() << "Field " << fieldName << " not found in type " << typeName;
    }
    Y_THROW_UNLESS(!Structs_.Layout().IsStatic(s));
    return static_cast<size_t>(Structs_.Layout().Offset(s));
}

size_t TJvmMetadata::FindTypeSize(std::string_view typeName) const {
    for (const void* t : Types_) {
        const char* tSn = Types_.Layout().StructName(t);
        if (tSn != nullptr && tSn == typeName) {
            return Types_.Layout().Size(t);
        }
    }
    throw yexception() << "Type " << typeName << " not found";
}

i32 TJvmMetadata::FindIntValue(std::string_view intName) const {
    for (const void* i : Ints_) {
        if (intName == Ints_.Layout().Name(i)) {
            return Ints_.Layout().Value(i);
        }
    }
    throw yexception() << "Int " << intName << " not found";
}

}
