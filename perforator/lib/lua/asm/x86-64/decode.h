#pragma once

#ifdef __GNUC__
#pragma GCC diagnostic push
#pragma GCC diagnostic ignored "-Wunused-parameter"
#endif
#include <contrib/libs/llvm18/lib/Target/X86/X86InstrInfo.h>
#ifdef __GNUC__
#pragma GCC diagnostic pop
#endif

#include <llvm/MC/MCAsmInfo.h>
#include <llvm/MC/MCContext.h>
#include <llvm/MC/MCDisassembler/MCDisassembler.h>
#include <llvm/MC/MCInst.h>
#include <llvm/MC/MCInstBuilder.h>
#include <llvm/MC/MCObjectFileInfo.h>
#include <llvm/MC/MCRegisterInfo.h>
#include <llvm/MC/MCSubtargetInfo.h>
#include <llvm/MC/TargetRegistry.h>
#include <llvm/Object/ELFObjectFile.h>
#include <llvm/Object/ObjectFile.h>
#include <llvm/Support/MemoryBuffer.h>
#include <llvm/Support/SourceMgr.h>
#include <llvm/Support/TargetSelect.h>
#include <llvm/Support/raw_ostream.h>
#include <llvm/Target/TargetMachine.h>

#include <library/cpp/logger/global/global.h>

#include <util/generic/array_ref.h>
#include <util/generic/function_ref.h>
#include <util/generic/hash.h>
#include <util/generic/maybe.h>
#include <util/generic/vector.h>

#include <perforator/lib/asm/evaluator.h>

namespace NPerforator::NLinguist::NLua::NAsm::NX86 {

TMaybe<i64> DecodeLuaClose(
    const llvm::Triple& triple, TConstArrayRef<ui8> bytecode);

TMaybe<i64> DecodeLuaOpenJit(
    const llvm::Triple& triple, TConstArrayRef<ui8> bytecode);

TMaybe<i64> DecodeLjDispatchUpdate(
    const llvm::Triple& triple, TConstArrayRef<ui8> bytecode);

}  // namespace NPerforator::NLinguist::NLua::NAsm::NX86
