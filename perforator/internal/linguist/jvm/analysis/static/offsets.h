#pragma once

#include <cstddef>
#include <cstdlib>

#include <optional>

#include <sys/types.h>

namespace NPerforator::NLinguist::NJvm {

struct TKindInfo {
    size_t CodeBlobKindOffset;
    unsigned char CodeBlobKindNmethod;

};

struct TOffsets {
    size_t CodeHeapNextSegmentOffset;

    std::optional<TKindInfo> KindInfo;

    int Version;

    // following two fields are offsets within actual stack frames, not within frame class
    ssize_t StackFrameReturnAddressOffset;
    ssize_t InterpreterStackFrameMethodOffset;

    static TOffsets Get();
};

} // namespace NPerforator::NLinguist::NJvm
