#include "../decode.h"

#ifdef __GNUC__
#pragma GCC diagnostic push
#pragma GCC diagnostic ignored "-Wunused-parameter"
#endif
// FIXME: This import is broken somehow
// #include <contrib/libs/llvm22/lib/Target/ARM/ARMInstrInfo.h>
#ifdef __GNUC__
#pragma GCC diagnostic pop
#endif


namespace NPerforator::NLinguist::NPython::NAsm {
    TMaybe<ThreadImageOffsetType> DecodePyThreadStateGetCurrent(
        [[maybe_unused]] const llvm::Triple& triple,
        [[maybe_unused]] TConstArrayRef<ui8> bytecode
    ) {
        return Nothing();
    }

    TMaybe<ThreadImageOffsetType> DecodeCurrentFastGet(
        [[maybe_unused]] const llvm::Triple& triple,
        [[maybe_unused]] TConstArrayRef<ui8> bytecode
    ) {
        return Nothing();
    }

    TMaybe<ui64> DecodePyGetVersion(
        [[maybe_unused]] const llvm::Triple& triple,
        [[maybe_unused]] ui64 functionAddress,
        [[maybe_unused]] TConstArrayRef<ui8> bytecode
    ) {
        return Nothing();
    }

    TMaybe<ui64> DecodeAutoTSSKeyAddress(
        [[maybe_unused]] const llvm::Triple& triple,
        [[maybe_unused]] ui64 pyGILStateEnsureAddress,
        [[maybe_unused]] TConstArrayRef<ui8> bytecode
    ) {
        return Nothing();
    }

    TMaybe<ui64> DecodeInterpHeadAddress(
        [[maybe_unused]] const llvm::Triple& triple,
        [[maybe_unused]] ui64 pyInterpreterStateHeadAddress,
        [[maybe_unused]] TConstArrayRef<ui8> bytecode
    ) {
        return Nothing();
    }
}
