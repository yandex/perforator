#include <perforator/internal/linguist/jvm/unwind/lib/offsets.h>

#include <library/cpp/json/json_value.h>
#include <library/cpp/json/json_writer.h>

int main() {
    using NPerforator::NLinguist::NJvm::TOffsets;
    TOffsets off = TOffsets::Get();
    NJson::TJsonValue spec;
    spec["CodeBlob::_kind"] = off.CodeBlobKindOffset;
    spec["CodeHeap::_next_segment"] = off.CodeHeapNextSegmentOffset;
    spec["frame::return_addr_offset"] = off.StackFrameReturnAddressOffset;
    spec["frame::interpreter_frame_method_offset"] = off.InterpreterStackFrameMethodOffset;
    Cout << NJson::WriteJson(spec) << Endl;
}
