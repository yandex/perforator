#pragma once

#include <string_view>
#include <cstddef>

#include <perforator/internal/linguist/jvm/analysis/api/api.h>

namespace NPerforator::NLinguist::NJvm {


// Information extracted from the VMStructs class.
// See https://github.com/openjdk/jdk/blob/89f9268ed7c2cb86891f23a10482cd459454bd32/src/hotspot/share/runtime/vmStructs.hpp#L34
struct TVMStructsAddresses {
    constexpr static std::string_view StructsAddressSym = "gHotSpotVMStructs";
    const void* StructsAddress;
    constexpr static std::string_view StructsStructNameOffsetSym = "gHotSpotVMStructEntryTypeNameOffset";
    const void* StructsStructNameOffset;
    constexpr static std::string_view StructsFieldNameOffsetSym = "gHotSpotVMStructEntryFieldNameOffset";
    const void* StructsFieldNameOffset;
    constexpr static std::string_view StructsTypeNameOffsetSym = "gHotSpotVMStructEntryTypeStringOffset";
    const void* StructsTypeNameOffset;
    constexpr static std::string_view StructsIsStaticOffsetSym = "gHotSpotVMStructEntryIsStaticOffset";
    const void* StructsIsStaticOffset;
    constexpr static std::string_view StructsOffsetOffsetSym = "gHotSpotVMStructEntryOffsetOffset";
    const void* StructsOffsetOffset;
    constexpr static std::string_view StructsAddressOffsetSym = "gHotSpotVMStructEntryAddressOffset";
    const void* StructsAddressOffset;
    constexpr static std::string_view StructsStrideSym = "gHotSpotVMStructEntryArrayStride";
    const void* StructsStride;

    constexpr static std::string_view TypesAddressSym = "gHotSpotVMTypes";
    const void* TypesAddress;
    constexpr static std::string_view TypesStructNameOffsetSym = "gHotSpotVMTypeEntryTypeNameOffset";
    const void* TypesStructNameOffset;
    constexpr static std::string_view TypesSuperNameOffsetSym = "gHotSpotVMTypeEntrySuperclassNameOffset";
    const void* TypesSuperNameOffset;
    constexpr static std::string_view TypesIsOopOffsetSym = "gHotSpotVMTypeEntryIsOopTypeOffset";
    const void* TypesIsOopOffset;
    constexpr static std::string_view TypesIsIntegerOffsetSym = "gHotSpotVMTypeEntryIsIntegerTypeOffset";
    const void* TypesIsIntegerOffset;
    constexpr static std::string_view TypesIsUnsignedOffsetSym = "gHotSpotVMTypeEntryIsUnsignedOffset";
    const void* TypesIsUnsignedOffset;
    constexpr static std::string_view TypesSizeOffsetSym = "gHotSpotVMTypeEntrySizeOffset";
    const void* TypesSizeOffset;
    constexpr static std::string_view TypesStrideSym = "gHotSpotVMTypeEntryArrayStride";
    const void* TypesStride;

    constexpr static std::string_view IntsAddressSym = "gHotSpotVMIntConstants";
    const void* IntsAddress;
    constexpr static std::string_view IntsNameOffsetSym = "gHotSpotVMIntConstantEntryNameOffset";
    const void* IntsNameOffset;
    constexpr static std::string_view IntsValueOffsetSym = "gHotSpotVMIntConstantEntryValueOffset";
    const void* IntsValueOffset;
    constexpr static std::string_view IntsStrideSym = "gHotSpotVMIntConstantEntryArrayStride";
    const void* IntsStride;
};

TJvmAnalysis ProcessDynamicLinkedJVM(TVMStructsAddresses addresses, ui32 version);

TJvmAnalysis ProcessJVMHeaders();


} // namespace NPerforator::NLinguist::NJvm
