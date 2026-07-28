#include "decode.h"

namespace NPerforator::NLinguist::NLua::NAsm::NArm {

TMaybe<i64> DecodeLuaClose(const llvm::Triple&, ui64, TConstArrayRef<ui8>) {
    // Not implemented yet
    return Nothing();
}

TMaybe<i64> DecodeLuaOpenJit(const llvm::Triple&, ui64, TConstArrayRef<ui8>) {
    // Not implemented yet
    return Nothing();
}

TMaybe<i64> DecodeLjDispatchUpdate(const llvm::Triple&, ui64, TConstArrayRef<ui8>) {
    // Not implemented yet
    return Nothing();
}

TMaybe<i64> DecodeLuaGc(const llvm::Triple&, ui64, TConstArrayRef<ui8>) {
    // Not implemented yet
    return Nothing();
}

TMaybe<i64> DecodeLjGcStep(const llvm::Triple&, ui64, TConstArrayRef<ui8>) {
    // Not implemented yet
    return Nothing();
}

} // namespace NPerforator::NLinguist::NLua::NAsm::NArm
