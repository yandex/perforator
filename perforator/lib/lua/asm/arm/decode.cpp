#include "decode.h"

namespace NPerforator::NLinguist::NLua::NAsm::NArm {

TMaybe<i64> DecodeLuaClose([[maybe_unused]] const llvm::Triple& triple,
    [[maybe_unused]] ui64 functionAddress,
    [[maybe_unused]] TConstArrayRef<ui8> bytecode) {
    // Not implemented yet
    return Nothing();
}

TMaybe<i64> DecodeLuaOpenJit([[maybe_unused]] const llvm::Triple& triple,
    [[maybe_unused]] ui64 functionAddress,
    [[maybe_unused]] TConstArrayRef<ui8> bytecode) {
    // Not implemented yet
    return Nothing();
}

TMaybe<i64> DecodeLjDispatchUpdate([[maybe_unused]] const llvm::Triple& triple,
    [[maybe_unused]] ui64 functionAddress,
    [[maybe_unused]] TConstArrayRef<ui8> bytecode) {
    // Not implemented yet
    return Nothing();
}

}  // namespace NPerforator::NLinguist::NLua::NAsm::NArm
