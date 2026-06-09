#include "offset_registry.h"

#include <util/stream/output.h>
#include <util/generic/yexception.h>


namespace NPerforator::NLinguist::NJvm {

TJvmMetadata::TJvmMetadata(
    std::span<const THotSpotStructEntry> structs,
    std::span<const THotSpotTypeEntry> types,
    const void* intsData,
    size_t intsCount,
    IntLayout intLayout
)
    : Structs_(structs)
    , Types_(types)
    , IntsData_(intsData)
    , IntsCount_(intsCount)
    , IntLayout_(intLayout)
{
}

const THotSpotStructEntry* TJvmMetadata::FindField(std::string_view typeName, std::string_view fieldName) const {
    for (const auto& s : Structs_) {
        if (s.TypeName == nullptr || s.FieldName == nullptr) {
            continue;
        }
        if (typeName == s.StructName && fieldName == s.FieldName) {
            return &s;
        }
    }
    return nullptr;
}

uintptr_t TJvmMetadata::FindStaticFieldAddress(std::string_view typeName, std::string_view fieldName) const {
    const THotSpotStructEntry* s = FindField(typeName, fieldName);
    if (s == nullptr) {
        throw yexception() << "Static field " << fieldName << " not found in type " << typeName;
    }
    Y_THROW_UNLESS(s->IsStatic);
    Y_THROW_UNLESS(s->Address != nullptr);
    return reinterpret_cast<uintptr_t>(s->Address);
}


size_t TJvmMetadata::FindFieldOffset(std::string_view typeName, std::string_view fieldName) const {
    const THotSpotStructEntry* s = FindField(typeName, fieldName);
    if (s == nullptr) {
        throw yexception() << "Field " << fieldName << " not found in type " << typeName;
    }
    Y_THROW_UNLESS(!s->IsStatic);
    return static_cast<size_t>(s->Offset);
}

size_t TJvmMetadata::FindTypeSize(std::string_view typeName) const {
    for (const THotSpotTypeEntry& t : Types_) {
        if (t.StructName != nullptr && t.StructName == typeName) {
            return t.Size;
        }
    }
    throw yexception() << "Type " << typeName << " not found";
}

i32 TJvmMetadata::FindIntValue(std::string_view intName) const {
    const void* elem = IntsData_;
    for (size_t i = 0; i < IntsCount_; elem = IntLayout_.Inc(elem), ++i) {
        if (intName == IntLayout_.Name(elem)) {
            return IntLayout_.Value(elem);
        }
    }
    throw yexception() << "Int " << intName << " not found";
}

}
