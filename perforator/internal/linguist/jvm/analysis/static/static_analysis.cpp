#include "static_analysis.h"

#include "offsets.h"

#include <perforator/internal/linguist/jvm/analysis/offset_registry/analyzer_impl.h>
#include <perforator/internal/linguist/jvm/analysis/offset_registry/offset_registry.h>

#include <util/stream/output.h>
#include <util/system/yassert.h>

namespace NPerforator::NLinguist::NJvm {

namespace {

size_t StructsLength(const char* ptr, TStructEntryLayout layout) {
    size_t length = 0;
    while (layout.StructName(ptr) != nullptr || layout.FieldName(ptr) != nullptr) {
        ptr += layout.Stride;
        ++length;
    }
    return length;
}

size_t TypesLength(const char* ptr, TTypeEntryLayout layout) {
    size_t length = 0;
    while (layout.StructName(ptr) != nullptr) {
        ptr += layout.Stride;
        ++length;
    }
    return length;
}

size_t IntsLength(const char* ptr, TIntEntryLayout layout) {
    size_t length = 0;
    while (layout.Name(ptr) != nullptr) {
        ptr += layout.Stride;
        ++length;
    }
    return length;
}

};

TJvmAnalysis ProcessJVMHeaders() {
    TJvmAnalysis analysis;

    TOffsets offsets = TOffsets::Get();

    if (offsets.KindInfo) {
        analysis.Cheatsheet.set_code_blob_kind(offsets.KindInfo->CodeBlobKindOffset);
        analysis.Cheatsheet.set_code_blob_kind_nmethod(static_cast<int>(offsets.KindInfo->CodeBlobKindNmethod));
    }
    analysis.Cheatsheet.set_code_heap_next_segment(offsets.CodeHeapNextSegmentOffset);
    analysis.Cheatsheet.set_nmethod_speculations(offsets.NmethodSpeculationsOffset);
    if (offsets.Version <= 24) {
        analysis.Cheatsheet.set_nmethod_jvmci_data_offset(offsets.NmethodJvmciDataOffset);
    }
    if (offsets.Version <= 21) {
        analysis.Cheatsheet.set_nmethod_scopes_data_addr(offsets.NmethodScopesDataBeginOffset);
    }

    analysis.Cheatsheet.set_frame_return_addr_offset(offsets.StackFrameReturnAddressOffset);
    analysis.Cheatsheet.set_frame_interpreter_frame_method_offset(offsets.InterpreterStackFrameMethodOffset);

    analysis.Version = static_cast<ui32>(offsets.Version);

    return analysis;
}

TJvmAnalysis ProcessDynamicLinkedJVM(TVMStructsAddresses addresses, ui32 version) {
    const auto* structs = *reinterpret_cast<char const* const*>(addresses.StructsAddress);
    const auto* types = *reinterpret_cast<char const* const*>(addresses.TypesAddress);
    const auto* ints = *reinterpret_cast<char const* const*>(addresses.IntsAddress);
    TStructEntryLayout structLayout{
        .Stride = *reinterpret_cast<uint64_t const*>(addresses.StructsStride),
        .StructNameOffset = *reinterpret_cast<uint64_t const*>(addresses.StructsStructNameOffset),
        .FieldNameOffset = *reinterpret_cast<uint64_t const*>(addresses.StructsFieldNameOffset),
        .TypeNameOffset = *reinterpret_cast<uint64_t const*>(addresses.StructsTypeNameOffset),
        .IsStaticOffset = *reinterpret_cast<uint64_t const*>(addresses.StructsIsStaticOffset),
        .OffsetOffset = *reinterpret_cast<uint64_t const*>(addresses.StructsOffsetOffset),
        .AddressOffset = *reinterpret_cast<uint64_t const*>(addresses.StructsAddressOffset),
    };
    TTypeEntryLayout typeLayout{
        .Stride = *reinterpret_cast<uint64_t const*>(addresses.TypesStride),
        .StructNameOffset = *reinterpret_cast<uint64_t const*>(addresses.TypesStructNameOffset),
        .SuperNameOffset = *reinterpret_cast<uint64_t const*>(addresses.TypesSuperNameOffset),
        .IsOopOffset = *reinterpret_cast<uint64_t const*>(addresses.TypesIsOopOffset),
        .IsIntegerOffset = *reinterpret_cast<uint64_t const*>(addresses.TypesIsIntegerOffset),
        .IsUnsignedOffset = *reinterpret_cast<uint64_t const*>(addresses.TypesIsUnsignedOffset),
        .SizeOffset = *reinterpret_cast<uint64_t const*>(addresses.TypesSizeOffset),
    };
    TIntEntryLayout intLayout{
        .Stride = *reinterpret_cast<uint64_t const*>(addresses.IntsStride),
        .NameOffset = *reinterpret_cast<uint64_t const*>(addresses.IntsNameOffset),
        .ValueOffset = *reinterpret_cast<uint64_t const*>(addresses.IntsValueOffset),
    };
    size_t structsLength = StructsLength(structs, structLayout);
    size_t typesLength = TypesLength(types, typeLayout);
    size_t intsLength = IntsLength(ints, intLayout);
    TJvmMetadata metadata{
        {structs, structsLength, structLayout},
        {types, typesLength, typeLayout},
        {ints, intsLength, intLayout},
    };
    TJvmAnalysis analysis = ProcessOffsetRegistry(metadata, TOffsetRegistryAnalysisOptions{}, version);
    analysis.Version = version;
    return analysis;
}

} // namespace NPerforator::NLinguist::NJvm
