#pragma once

#ifdef __GNUC__
#pragma GCC diagnostic push
#pragma GCC diagnostic ignored "-Wunused-parameter"
#endif
#include <contrib/libs/llvm18/lib/Target/X86/X86InstrInfo.h>
#ifdef __GNUC__
#pragma GCC diagnostic pop
#endif

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

#include <util/generic/array_ref.h>
#include <util/generic/function_ref.h>
#include <util/generic/maybe.h>

namespace NPerforator::NLinguist::NPython::NDecode {
    using ThreadImageOffsetType = i64;
} // namespace NPerforator::NLinguist::NPython::NDecode


namespace NPerforator::NLinguist::NPython::NDecode::NX86 {

template <typename ELFT>
void DecodeInstructions(
    llvm::object::ELFObjectFile<ELFT>* elf,
    TConstArrayRef<ui8> bytecode,
    TFunctionRef<bool(const llvm::MCInst&, ui64 size)> instCallback
) {
    LLVMInitializeX86Target();
    LLVMInitializeX86TargetInfo();
    LLVMInitializeX86TargetMC();
    LLVMInitializeX86Disassembler();

    auto triple = elf->makeTriple();
    std::string error;
    const llvm::Target* target = llvm::TargetRegistry::lookupTarget(triple.getTriple(), error);
    if (!target) {
        Cout << "Failed to lookup target by triple " << triple.getTriple() << ' ' << error << Endl;
        return;
    }

    auto mri = target->createMCRegInfo(triple.getTriple());

    llvm::MCTargetOptions options;
    std::unique_ptr<llvm::MCAsmInfo> asmInfo(
        target->createMCAsmInfo(
            *mri,
            triple.getTriple(),
            options
        )
    );

    auto subTargetInfo = target->createMCSubtargetInfo(triple.getTriple(), "", "");
    if (subTargetInfo == nullptr) {
        return;
    }

    llvm::MCContext context(triple, asmInfo.get(), mri, subTargetInfo);

    std::unique_ptr<llvm::MCDisassembler> disasm(target->createMCDisassembler(*subTargetInfo, context));

    for (size_t i = 0; i < bytecode.size(); ) {
        llvm::MCInst inst;
        ui64 size = 0;
        auto status = disasm->getInstruction(inst, size, llvm::ArrayRef<ui8>(bytecode.data() + i, bytecode.size() - i), i, llvm::nulls());

        if (status != llvm::MCDisassembler::Success) {
            break;
        }

        if (llvm::X86::isRET(inst.getOpcode()) ||
            llvm::X86::isRETFQ(inst.getOpcode()) ||
            llvm::X86::isRETF(inst.getOpcode())) {
            break;
        }

        if (!instCallback(inst, size)) {
            break;
        }

        // Move to the next instruction
        i += size;
    }
}

/*
000000000028a0b0 <_PyThreadState_GetCurrent@@Base>:
  28a0b0:       f3 0f 1e fa             endbr64
  28a0b4:       64 48 8b 04 25 f8 ff    mov    %fs:0xfffffffffffffff8,%rax
  28a0bb:       ff ff
  28a0bd:       c3                      ret
  28a0be:       66 90                   xchg   %ax,%ax

Parse thread image offset for `_Py_tss_tstate`. In this case it is -8 (0xfffffffffffffff8).
Look for mov %fs:<offset>,... and extract <offset>
*/
template <typename ELFT>
TMaybe<ThreadImageOffsetType> DecodePyThreadStateGetCurrent(
    llvm::object::ELFObjectFile<ELFT>* elf,
    TConstArrayRef<ui8> bytecode
) {
    ThreadImageOffsetType result = 0;
    DecodeInstructions(elf, bytecode, [&](const llvm::MCInst& inst, ui64 size) {
        Y_UNUSED(size);

        switch (inst.getOpcode()) {
        // Parse `mov %fs:0xfffffffffffffff8,%rax`
        case llvm::X86::MOV64rm:
        case llvm::X86::MOV32rm:
            bool foundFSorGSRegister = false;
            ThreadImageOffsetType lastNegativeImm = 0;

            for (size_t j = 0; j < inst.getNumOperands(); j++) {
                const llvm::MCOperand& operand = inst.getOperand(j);
                // Look for negative imms because TLS is located to the left from %fs
                if (operand.isImm() && operand.getImm() < 0) {
                    lastNegativeImm = operand.getImm();
                }
                if (operand.isReg() && (operand.getReg() == llvm::X86::FS || operand.getReg() == llvm::X86::GS)) {
                    foundFSorGSRegister = true;
                }
            }

            if (foundFSorGSRegister && lastNegativeImm < 0) {
                result = lastNegativeImm;
            }
        }
        return true;
    });

    if (result == 0) {
        return Nothing();
    }

    return MakeMaybe(result);
}

/*
0000000001db7c50 <current_fast_get>:
 1db7c50:       55                      push   %rbp
 1db7c51:       48 89 e5                mov    %rsp,%rbp
 1db7c54:       48 89 7d f8             mov    %rdi,-0x8(%rbp)
 1db7c58:       64 48 8b 04 25 00 00    mov    %fs:0x0,%rax
 1db7c5f:       00 00
 1db7c61:       48 8d 80 f8 ff ff ff    lea    -0x8(%rax),%rax
 1db7c68:       48 8b 00                mov    (%rax),%rax
 1db7c6b:       5d                      pop    %rbp
 1db7c6c:       c3                      retq

 Parse thread image offset for `_Py_tss_tstate`. In this case it is -8 (0xfffffffffffffff8).
 Look for lea instruction and extract offset from it.
 We do not want to provide general code here because it seems hard and unnecessary yet.
 Though later if we find something which can not be disassembled this way, we can give more general approach a try.
 For example: perform some graph calculations where vertices are registers and edges are mov's or lea's
    to extract certain %fs offset.
*/
template <typename ELFT>
TMaybe<ThreadImageOffsetType> DecodeCurrentFastGet(
    llvm::object::ELFObjectFile<ELFT>* elf,
    TConstArrayRef<ui8> bytecode
) {
    ThreadImageOffsetType lastNegativeImm = 0;
    DecodeInstructions(elf, bytecode, [&](const llvm::MCInst& inst, ui64 size) {
        Y_UNUSED(size);

        switch (inst.getOpcode()) {
        case llvm::X86::LEA64r:
        case llvm::X86::LEA64_32r:
        case llvm::X86::LEA32r:
            ThreadImageOffsetType negativeImm = 0;
            bool foundUnfamiliarRegisters = false;

            for (size_t j = 0; j < inst.getNumOperands(); j++) {
                const llvm::MCOperand& operand = inst.getOperand(j);
                if (operand.isImm() && operand.getImm() < 0) {
                    negativeImm = operand.getImm();
                }
                if (operand.isReg() &&
                    operand.getReg() != llvm::X86::NoRegister &&
                    operand.getReg() != llvm::X86::FS &&
                    operand.getReg() != llvm::X86::RAX &&
                    operand.getReg() != llvm::X86::EAX &&
                    operand.getReg() != llvm::X86::GS
                ) {
                    foundUnfamiliarRegisters = true;
                }
            }

            if (!foundUnfamiliarRegisters && negativeImm < 0) {
                lastNegativeImm = negativeImm;
            }
        }
        return true;
    });

    if (lastNegativeImm == 0) {
        return Nothing();
    }

    return MakeMaybe(lastNegativeImm);
}

/*
 * Disassembles Py_GetVersion function to find the address of PY_VERSION string.
 * Example of Py_GetVersion assembly:

00000000001d3330 <Py_GetVersion@@Base>:
  1d3330:       f3 0f 1e fa             endbr64
  1d3334:       53                      push   %rbx
  1d3335:       e8 a6 a0 02 00          callq  1fd3e0 <Py_GetCompiler@@Base>
  1d333a:       48 89 c3                mov    %rax,%rbx
  1d333d:       e8 7e 42 fb ff          callq  1875c0 <Py_GetBuildInfo@@Base>
  1d3342:       49 89 d9                mov    %rbx,%r9
  1d3345:       be fa 00 00 00          mov    $0xfa,%esi
  1d334a:       48 8d 0d 47 dd 02 00    lea    0x2dd47(%rip),%rcx        # 201098 <_IO_stdin_used@@Base+0x3098>
  1d3351:       49 89 c0                mov    %rax,%r8
  1d3354:       48 8d 15 1f 64 06 00    lea    0x6641f(%rip),%rdx        # 23977a <_PyUnicode_TypeRecords@@Base+0xd01a>
  1d335b:       48 8d 3d de 58 16 00    lea    0x1658de(%rip),%rdi        # 338c40 <Py_FileSystemDefaultEncoding@@Base+0x3d0>
  1d3362:       31 c0                   xor    %eax,%eax
  1d3364:       e8 27 7f fa ff          callq  17b290 <PyOS_snprintf@@Base>
  1d3369:       48 8d 05 d0 58 16 00    lea    0x1658d0(%rip),%rax        # 338c40 <Py_FileSystemDefaultEncoding@@Base+0x3d0>
  1d3370:       5b                      pop    %rbx
  1d3371:       c3                      retq
  1d3372:       66 2e 0f 1f 84 00 00    nopw   %cs:0x0(%rax,%rax,1)
  1d3379:       00 00 00
  1d337c:       0f 1f 40 00             nopl   0x0(%rax)

* Another example:

00000000047cb400 <Py_GetVersion@@Base>:
 47cb400:       53                      push   %rbx
 47cb401:       e8 3a 00 00 00          callq  47cb440 <Py_GetBuildInfo@@Base>
 47cb406:       48 89 c3                mov    %rax,%rbx
 47cb409:       e8 92 00 00 00          callq  47cb4a0 <Py_GetCompiler@@Base>
 47cb40e:       bf 20 35 30 0b          mov    $0xb303520,%edi
 47cb413:       be fa 00 00 00          mov    $0xfa,%esi
 47cb418:       ba 27 e9 7f 01          mov    $0x17fe927,%edx
 47cb41d:       b9 32 c5 7b 01          mov    $0x17bc532,%ecx
 47cb422:       49 89 d8                mov    %rbx,%r8
 47cb425:       49 89 c1                mov    %rax,%r9
 47cb428:       31 c0                   xor    %eax,%eax
 47cb42a:       e8 51 51 00 00          callq  47d0580 <PyOS_snprintf@@Base>
 47cb42f:       b8 20 35 30 0b          mov    $0xb303520,%eax
 47cb434:       5b                      pop    %rbx
 */
template <typename ELFT>
TMaybe<ui64> DecodePyGetVersion(
    llvm::object::ELFObjectFile<ELFT>* elf,
    ui64 functionAddress,
    TConstArrayRef<ui8> bytecode
) {
    TMaybe<ui64> pythonVersionBuffer;
    ui64 rip = functionAddress;

    // Look for instructions that load address into the 4th argument argument register (rcx/ecx)
    // Check the implementation of Py_GetVersion: https://github.com/python/cpython/blob/v3.11.0/Python/getversion.c#L12
    DecodeInstructions(elf, bytecode, [&](const llvm::MCInst& inst, ui64 size) {
        rip += size;

        switch (inst.getOpcode()) {
            // Handle absolute address loading via MOV
            case llvm::X86::MOV32ri:
            case llvm::X86::MOV64ri: {
                // Check if destination is rcx or ecx (4th argument)
                if (inst.getOperand(0).isReg() &&
                     (inst.getOperand(0).getReg() == llvm::X86::ECX ||
                     inst.getOperand(0).getReg() == llvm::X86::RCX)) {

                    // Get immediate value (absolute address)
                    if (inst.getOperand(1).isImm()) {
                        pythonVersionBuffer = static_cast<ui64>(inst.getOperand(1).getImm());
                        return false;
                    }
                }
                break;
            }

            // Handle RIP-relative address loading via LEA
            case llvm::X86::LEA64r:
            case llvm::X86::LEA32r: {
                bool isTargetReg = false;
                bool foundRIP = false;
                i64 offset = 0;

                // Check if destination is rcx/ecx (4th argument)
                if (inst.getOperand(0).isReg() &&
                    (inst.getOperand(0).getReg() == llvm::X86::RCX ||
                     inst.getOperand(0).getReg() == llvm::X86::ECX)) {
                    isTargetReg = true;
                }

                // Look for RIP-relative addressing
                for (size_t j = 0; j < inst.getNumOperands(); j++) {
                    const llvm::MCOperand& operand = inst.getOperand(j);
                    if (operand.isReg() && operand.getReg() == llvm::X86::RIP) {
                        foundRIP = true;
                    }
                    if (operand.isImm()) {
                        offset = static_cast<i64>(operand.getImm());
                    }
                }

                if (isTargetReg && foundRIP && offset != 0) {
                    pythonVersionBuffer = rip + offset;
                    return false;
                }
                break;
            }
        }

        return true;
    });

    return pythonVersionBuffer;
}

} // namespace NPerforator::NLinguist::NPython::NDecode::NX86
