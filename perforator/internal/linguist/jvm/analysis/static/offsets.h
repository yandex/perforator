#pragma once

#include <perforator/internal/linguist/jvm/analysis/api/api.h>

#include <cstddef>
#include <cstdlib>

#include <util/system/types.h>

#include <optional>
#include <string_view>

namespace NPerforator::NLinguist::NJvm {

struct TKindInfo {
    size_t CodeBlobKindOffset;
    unsigned char CodeBlobKindNmethod;
};



struct TOffsets {
    size_t CodeHeapNextSegmentOffset;

    std::optional<TKindInfo> KindInfo;

    NPerforator::NBinaryProcessing::NJvm::Version Version;

    size_t NmethodSpeculationsOffset;
    size_t NmethodJvmciDataOffset;
    size_t NmethodScopesDataBeginOffset;

    // following two fields are offsets within actual stack frames, not within frame class
    i32 StackFrameReturnAddressOffset;
    i32 InterpreterStackFrameMethodOffset;

    // TODO: ideally we should detect version rather than take it as input.
    static TOffsets Get(std::string_view path, NPerforator::NBinaryProcessing::NJvm::Version version);
};

} // namespace NPerforator::NLinguist::NJvm
