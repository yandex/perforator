#include "offsets.h"

#include "parser.h"

#include <util/generic/yexception.h>

namespace NPerforator::NLinguist::NJvm {
namespace {

using enum llvm::dwarf::Attribute;


TOffsets Parse(const NDwarfParser::TParser& parser, ui32 version) {
    TOffsets off;

    off.Version = version;

    std::string nmethodUnit = "nmethod.cpp";
    std::string codeHeapUnit = "memory/heap.cpp";
    std::string frameUnit = "runtime/frame.cpp";
    std::string compiledMethodUnit = "code/compiledMethod.cpp";
    std::string codeBlobUnit = "codeBlob.cpp";

    off.InterpreterStackFrameMethodOffset = NDwarfParser::ReadI32Attr(
        parser.Resolve({.Unit=frameUnit, .QName={"frame", "", "interpreter_frame_method_offset"}}),
        DW_AT_const_value
    );
    off.StackFrameReturnAddressOffset = NDwarfParser::ReadI32Attr(
        parser.Resolve({.Unit=frameUnit, .QName={"frame", "", "return_addr_offset"}}),
        DW_AT_const_value
    );

    if (version >= 21 && version <= 24) {
        off.NmethodJvmciDataOffset = NDwarfParser::ReadSizeTAttr(
            parser.Resolve({.Unit=nmethodUnit, .QName={"nmethod", "_jvmci_data_offset"}}),
            DW_AT_data_member_location
        );
    } else {
        off.NmethodJvmciDataOffset = SIZE_MAX;
    }
    if (version <= 21) {
        off.NmethodScopesDataBeginOffset = NDwarfParser::ReadSizeTAttr(
            parser.Resolve({.Unit=compiledMethodUnit, .QName={"CompiledMethod", "_scopes_data_begin"}}),
            DW_AT_data_member_location
        );
    } else {
        off.NmethodScopesDataBeginOffset = SIZE_MAX;
    }
    off.NmethodSpeculationsOffset = NDwarfParser::ReadSizeTAttr(
        parser.Resolve({.Unit=nmethodUnit, .QName={"nmethod", "_speculations_offset"}}),
        DW_AT_data_member_location
    );
    off.CodeHeapNextSegmentOffset = NDwarfParser::ReadSizeTAttr(
        parser.Resolve({.Unit=codeHeapUnit, .QName={"CodeHeap", "_next_segment"}}),
        DW_AT_data_member_location
    );

    if (version > 21 && version <= 24) {
        off.KindInfo.emplace();
        off.KindInfo->CodeBlobKindOffset = NDwarfParser::ReadSizeTAttr(
            parser.Resolve({.Unit=codeBlobUnit, .QName={"CodeBlob", "_kind"}}),
            DW_AT_data_member_location
        );
        off.KindInfo->CodeBlobKindNmethod = NDwarfParser::ReadU8Attr(
            parser.Resolve({.Unit=codeBlobUnit, .QName={"CodeBlobKind", "Nmethod"}}),
            DW_AT_const_value
        );
    }
    return off;
}
}

TOffsets TOffsets::Get(std::string_view path, ui32 version) {
    NDwarfParser::TParser parser(path);

    return Parse(parser, version);
}

} // namespace NPerforator::NLinguist::NJvm
