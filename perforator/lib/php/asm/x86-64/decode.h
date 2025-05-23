#pragma once

#ifdef __GNUC__
    #pragma GCC diagnostic push
    #pragma GCC diagnostic ignored "-Wunused-parameter"
#endif
#include <contrib/libs/llvm18/lib/Target/X86/X86InstrInfo.h>
#ifdef __GNUC__
    #pragma GCC diagnostic pop
#endif

#include <llvm/TargetParser/Triple.h>
#include <llvm/Support/TargetSelect.h>
#include <llvm/MC/MCDisassembler/MCDisassembler.h>
#include <llvm/MC/MCContext.h>
#include <llvm/MC/MCInst.h>
#include <llvm/MC/MCRegisterInfo.h>
#include <llvm/MC/MCSubtargetInfo.h>
#include <llvm/MC/MCAsmInfo.h>
#include <llvm/Support/MemoryBuffer.h>
#include <llvm/Support/SourceMgr.h>
#include <llvm/Support/raw_ostream.h>
#include <llvm/Target/TargetMachine.h>
#include <llvm/MC/MCInstBuilder.h>
#include <llvm/MC/MCObjectFileInfo.h>
#include <llvm/MC/TargetRegistry.h>
#include <llvm/Object/ELFObjectFile.h>
#include <llvm/Object/ObjectFile.h>

#include <util/generic/maybe.h>
#include <util/system/types.h>

namespace NPerforator::NLinguist::NPhp::NAsm::NX86 {

TMaybe<ui64> DecodePhpVersion(
    const llvm::Triple& triple,
    ui64 functionAddress,
    TConstArrayRef<ui8> bytecode
);

TMaybe<ui64> DecodeZmInfoPhpCore(
    const llvm::Triple& triple,
    ui64 functionAddress,
    TConstArrayRef<ui8> bytecode
);

TMaybe<ui64> DecodeZendVmKind(
    const llvm::Triple& triple,
    ui64 functionAddress,
    TConstArrayRef<ui8> bytecode
);

} // namespace NPerforator::NLinguist::NPhp::NAsm::NX86
