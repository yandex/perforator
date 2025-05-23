#include <cstddef>
#include <cstdlib>

namespace NPerforator::NLinguist::NJvm {

struct TOffsets {
    size_t CodeHeapNextSegmentOffset;
    size_t CodeBlobKindOffset;

    // following two fields are offsets within actual stack frames, not within frame class
    ssize_t StackFrameReturnAddressOffset;
    ssize_t InterpreterStackFrameMethodOffset;

    static TOffsets Get();
};

} // namespace NPerforator::NLinguist::NJvm
