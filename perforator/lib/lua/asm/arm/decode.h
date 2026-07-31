#pragma once

#include <llvm/TargetParser/Triple.h>

#include <util/generic/maybe.h>
#include <util/generic/array_ref.h>

namespace NPerforator::NLinguist::NLua::NAsm::NArm {

TMaybe<i64> DecodeLuaClose(const llvm::Triple& triple, ui64 functionAddress, TConstArrayRef<ui8> bytecode);

TMaybe<i64> DecodeLuaOpenJit(const llvm::Triple& triple, ui64 functionAddress, TConstArrayRef<ui8> bytecode);

TMaybe<i64> DecodeLjDispatchUpdate(const llvm::Triple& triple, ui64 functionAddress, TConstArrayRef<ui8> bytecode);

TMaybe<i64> DecodeLuaGc(const llvm::Triple& triple, ui64 functionAddress, TConstArrayRef<ui8> bytecode);

TMaybe<i64> DecodeLjGcStep(const llvm::Triple& triple, ui64 functionAddress, TConstArrayRef<ui8> bytecode);

} // namespace NPerforator::NLinguist::NLua::NAsm::NArm
