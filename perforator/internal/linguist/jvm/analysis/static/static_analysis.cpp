#include "static_analysis.h"

#include "offsets.h"

#include <perforator/internal/linguist/jvm/analysis/offset_registry/analyzer_impl.h>
#include <perforator/internal/linguist/jvm/analysis/offset_registry/offset_registry.h>

#include <util/stream/output.h>
#include <util/system/yassert.h>

#include <span>


namespace NPerforator::NLinguist::NJvm {

namespace {

size_t StructsLength(const THotSpotStructEntry* entries) {
    size_t length = 0;
    while (entries[length].StructName != nullptr || entries[length].FieldName != nullptr) {
        ++length;
    }
    return length;
}

size_t TypesLength(const THotSpotTypeEntry* entries) {
    size_t length = 0;
    while (entries[length].StructName != nullptr) {
        ++length;
    }
    return length;
}

size_t IntsLength(const void* entries, IntLayout layout) {
    size_t length = 0;
    while (layout.Name(entries) != nullptr) {
        entries = layout.Inc(entries);
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
    const auto* structs = *reinterpret_cast<THotSpotStructEntry const* const*>(addresses.StructsAddress);
    const auto* types = *reinterpret_cast<THotSpotTypeEntry const* const*>(addresses.TypesAddress);
    const auto* ints = *reinterpret_cast<void const* const*>(addresses.IntsAddress);
    IntLayout intLayout{
        .Stride = *reinterpret_cast<uint64_t const*>(addresses.IntsStride),
        .NameOffset = *reinterpret_cast<uint64_t const*>(addresses.IntsNameOffset),
        .ValueOffset = *reinterpret_cast<uint64_t const*>(addresses.IntsValueOffset),
    };
    size_t structsLength = StructsLength(structs);
    size_t typesLength = TypesLength(types);
    size_t intsLength = IntsLength(ints, intLayout);
    TJvmMetadata metadata{
        std::span<const THotSpotStructEntry>(structs, structsLength),
        std::span<const THotSpotTypeEntry>(types, typesLength),
        ints,
        intsLength,
        intLayout,
    };
    TJvmAnalysis analysis = ProcessOffsetRegistry(metadata, TOffsetRegistryAnalysisOptions{}, version);
    analysis.Version = version;
    return analysis;
}

} // namespace NPerforator::NLinguist::NJvm
