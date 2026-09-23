#include "../decode.h"

#ifdef __GNUC__
#pragma GCC diagnostic push
#pragma GCC diagnostic ignored "-Wunused-parameter"
#endif
#include <contrib/libs/llvm22/lib/Target/X86/X86InstrInfo.h>
#ifdef __GNUC__
#pragma GCC diagnostic pop
#endif

#include <llvm/MC/MCInst.h>

#include <util/stream/format.h>

#include <library/cpp/logger/log.h>
#include <library/cpp/logger/global/global.h>

#include <perforator/lib/asm/evaluator.h>
#include <perforator/lib/pthread/pthread.h>

namespace NPerforator::NPthread::NAsm {

/*
Disassembles bytecode of pthread_getspecific function and returns access info to TSS.

Example bytecode:
0000000000010cb0 <__pthread_getspecific>:
   10cb0:       f3 0f 1e fa             endbr64
   10cb4:       83 ff 1f                cmp    $0x1f,%edi
   10cb7:       77 3f                   ja     10cf8 <__pthread_getspecific+0x48>
   10cb9:       89 f8                   mov    %edi,%eax
   10cbb:       48 83 c0 31             add    $0x31,%rax
   10cbf:       48 c1 e0 04             shl    $0x4,%rax
   10cc3:       64 48 8b 14 25 10 00    mov    %fs:0x10,%rdx
   10cca:       00 00
   10ccc:       48 01 d0                add    %rdx,%rax
   10ccf:       4c 8b 40 08             mov    0x8(%rax),%r8
   10cd3:       4d 85 c0                test   %r8,%r8
   10cd6:       74 16                   je     10cee <__pthread_getspecific+0x3e>
   10cd8:       89 ff                   mov    %edi,%edi
   10cda:       48 8d 15 5f d6 00 00    lea    0xd65f(%rip),%rdx        # 1e340 <__GI___pthread_keys>
   10ce1:       48 8b 30                mov    (%rax),%rsi
   10ce4:       48 c1 e7 04             shl    $0x4,%rdi
   10ce8:       48 39 34 3a             cmp    %rsi,(%rdx,%rdi,1)
   10cec:       75 42                   jne    10d30 <__pthread_getspecific+0x80>
   10cee:       4c 89 c0                mov    %r8,%rax
   10cf1:       c3                      retq
   10cf2:       66 0f 1f 44 00 00       nopw   0x0(%rax,%rax,1)
   10cf8:       81 ff ff 03 00 00       cmp    $0x3ff,%edi
   10cfe:       77 40                   ja     10d40 <__pthread_getspecific+0x90>
   10d00:       89 f9                   mov    %edi,%ecx
   10d02:       89 f8                   mov    %edi,%eax
   10d04:       83 e1 1f                and    $0x1f,%ecx
   10d07:       c1 e8 05                shr    $0x5,%eax
   10d0a:       64 48 8b 14 c5 10 05    mov    %fs:0x510(,%rax,8),%rdx
   10d11:       00 00
   10d13:       49 89 d0                mov    %rdx,%r8
   10d16:       48 85 d2                test   %rdx,%rdx
   10d19:       74 d3                   je     10cee <__pthread_getspecific+0x3e>
   10d1b:       89 c8                   mov    %ecx,%eax
   10d1d:       48 c1 e0 04             shl    $0x4,%rax
   10d21:       48 01 d0                add    %rdx,%rax
   10d24:       eb a9                   jmp    10ccf <__pthread_getspecific+0x1f>
   10d26:       66 2e 0f 1f 84 00 00    nopw   %cs:0x0(%rax,%rax,1)
   10d2d:       00 00 00
   10d30:       48 c7 40 08 00 00 00    movq   $0x0,0x8(%rax)
   10d37:       00
   10d38:       45 31 c0                xor    %r8d,%r8d
   10d3b:       eb b1                   jmp    10cee <__pthread_getspecific+0x3e>
   10d3d:       0f 1f 00                nopl   (%rax)
   10d40:       45 31 c0                xor    %r8d,%r8d
   10d43:       eb a9                   jmp    10cee <__pthread_getspecific+0x3e>
   10d45:       66 2e 0f 1f 84 00 00    nopw   %cs:0x0(%rax,%rax,1)
   10d4c:       00 00 00
   10d4f:       90                      nop
*/
std::expected<TAccessTSSInfo, EDecodePthreadGetspecificError> DecodePthreadGetspecific(
    const llvm::Triple& triple,
    TConstArrayRef<ui8> bytecode
) {
    TAccessTSSInfo result;
    bool foundKeyDataSize = false;

    enum class EAnalysisState {
        Start,
        AfterFirstCmp,     // After checking key < PTHREAD_KEY_2NDLEVEL_SIZE
        AfterSecondCmp     // After checking key < PTHREAD_KEYS_MAX
    };

    EAnalysisState state = EAnalysisState::Start;

    auto error = NPerforator::NAsm::DecodeInstructions(TLoggerOperator<TGlobalLog>::Log(), triple, bytecode, [&](const llvm::MCInst& inst, ui64 size) {
        Y_UNUSED(size);

        if (state == EAnalysisState::AfterSecondCmp &&
            result.SpecificArrayOffset != 0 &&
            NPerforator::NAsm::IsPassControlFlow(inst)) {
            return false;
        }

        switch (inst.getOpcode()) {
            case llvm::X86::CMP32ri8:
            case llvm::X86::CMP32ri:
            {
                if (inst.getNumOperands() >= 2 && inst.getOperand(1).isImm()) {
                    i64 immValue = inst.getOperand(1).getImm();
                    // First comparison usually checks PTHREAD_KEY_2NDLEVEL_SIZE
                    if (state == EAnalysisState::Start && immValue > 0) {
                        result.KeySecondLevelSize = immValue + 1; // +1 because it's a "less than or equal" comparison
                        state = EAnalysisState::AfterFirstCmp;
                    }
                    // Second comparison after branch usually checks PTHREAD_KEYS_MAX
                    else if (state == EAnalysisState::AfterFirstCmp && ui64(immValue) > result.KeySecondLevelSize) {
                        result.KeysMax = immValue + 1; // +1 because it's a "less than or equal" comparison
                        state = EAnalysisState::AfterSecondCmp;
                    }
                }
                break;
            }

            // Access to TLS via %fs or %gs - Extract TLS offsets
            case llvm::X86::MOV64rm:
            case llvm::X86::MOV32rm:
            {
                // Example: mov %fs:0x510(,%rax,8),%rdx
                // Operand 0: Destination register
                // Operand 1: Base register (empty in our case)
                // Operand 2: Scale (8 in our example)
                // Operand 3: Index register (RAX in our example)
                // Operand 4: Displacement (0x510 in our example)
                // Operand 5: Segment register (%fs in our example)

                bool foundFSorGSAccess = (
                    inst.getNumOperands() >= 6 &&
                    inst.getOperand(5).isReg() &&
                    (inst.getOperand(5).getReg() == llvm::X86::FS || inst.getOperand(5).getReg() == llvm::X86::GS)
                );
                i64 disposition = (inst.getNumOperands() >= 5) ? inst.getOperand(4).getImm() : 0;
                bool hasArrayIndexing = (inst.getNumOperands() >= 4 && inst.getOperand(3).isReg() && inst.getOperand(3).getReg() != llvm::X86::NoRegister);

                if (foundFSorGSAccess && hasArrayIndexing) {
                    // mov %fs:0x510(,%rax,8),%rdx   from our example
                    result.SpecificArrayOffset = disposition;
                } else if (foundFSorGSAccess) {
                    // mov %fs:0x10,%rdx   from our example
                    result.StructPthreadPointerOffset = disposition;
                } else if (state == EAnalysisState::AfterFirstCmp && result.PthreadKeyData.Size != 0 && disposition > 0) {
                    // mov 0x8(%rax),%r8 from our example
                    result.PthreadKeyData.ValueOffset = disposition;
                }

                break;
            }

            // Left shift operation (SHL) - used to calculate the size of pthread_key_data structure
            case llvm::X86::SHL64ri:
            case llvm::X86::SHL32ri:
            {
                // Only process the first SHL we encounter
                if (!foundKeyDataSize && inst.getNumOperands() == 3 && inst.getOperand(2).isImm()) {
                    i64 shiftValue = inst.getOperand(2).getImm();
                    result.PthreadKeyData.Size = (1 << shiftValue);
                    foundKeyDataSize = true;
                }
                break;
            }
        }

        return true;
    });

    if (error != NPerforator::NAsm::EDecodeInstructionError::NoError) {
        return std::unexpected(EDecodePthreadGetspecificError::FailedToDecodeInstructions);
    }

    if (!foundKeyDataSize) {
        return std::unexpected(EDecodePthreadGetspecificError::NoPthreadKeyDataSize);
    }

    if (result.PthreadKeyData.ValueOffset == 0) {
        return std::unexpected(EDecodePthreadGetspecificError::NoPthreadKeyDataValueOffset);
    }

    result.PthreadKeyData.SeqOffset = 0;

    if (result.KeysMax == 0) {
        return std::unexpected(EDecodePthreadGetspecificError::NoPthreadKeysMax);
    }

    if (result.KeySecondLevelSize == 0) {
        return std::unexpected(EDecodePthreadGetspecificError::NoPthreadKeySecondLevelSize);
    }

    result.KeyFirstLevelSize = result.KeysMax / result.KeySecondLevelSize;

    if (result.SpecificArrayOffset == 0) {
        return std::unexpected(EDecodePthreadGetspecificError::NoPthreadKeySpecificArrayOffset);
    }

    // Calculate specific_1stblock offset from specific, as they are adjacent in memory
    result.FirstSpecificBlockOffset = result.SpecificArrayOffset - result.KeySecondLevelSize * result.PthreadKeyData.Size;

    if (result.FirstSpecificBlockOffset >= result.SpecificArrayOffset) {
        return std::unexpected(EDecodePthreadGetspecificError::NoPthreadKeySpecific1stBlockOffset);
    }

    return result;
}

} // namespace NPerforator::NPthread::NAsm::NX86
