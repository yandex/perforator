#include "decode.h"

#include <perforator/lib/asm/x86/evaluator.h>

namespace NPerforator::NLinguist::NPhp::NAsm::NX86 {

/*

00000000006be0fb <php_version>:
    6be0fb:   f3 0f 1e fa             endbr64
    6be0ff:   55                      push   %rbp
    6be100:   48 89 e5                mov    %rsp,%rbp
    6be103:   48 8d 05 fb 1e 09 01    lea    0x1091efb(%rip),%rax        # 1750005 <long_min_digits+0x135> <-- we need to receive this
    6be10a:   5d                      pop    %rbp
    6be10b:   c3                      ret

*/

TMaybe<ui64> DecodePhpVersion(const llvm::Triple& triple, ui64 functionAddress, TConstArrayRef<ui8> bytecode) {
    auto instructionEvaluator = NPerforator::NAsm::NX86::MakeDefaultInstructionEvaluator();
    NPerforator::NAsm::NX86::TBytecodeEvaluator evaluator(
        triple,
        NPerforator::NAsm::NX86::MakeInitialState(functionAddress),
        bytecode,
        *instructionEvaluator,
        NPerforator::NAsm::NX86::MakeStopOnRetCondition());
    auto result = evaluator.Evaluate();
    if (!result.has_value()) {
        return Nothing();
    }

    auto raxValue = GetRegisterValueOrAddress(result->State, llvm::X86::RAX);
    if (raxValue) {
        return raxValue;
    }

    auto eaxValue = GetRegisterValueOrAddress(result->State, llvm::X86::EAX);
    if (eaxValue) {
        return eaxValue;
    }

    return Nothing();
}

/*

000000000021ff8d <zm_info_php_core>:
    21ff8d:   f3 0f 1e fa             endbr64
    21ff91:   55                      push   %rbp
    21ff92:   48 89 fd                mov    %rdi,%rbp
    21ff95:   e8 03 7a ff ff          call   21799d <php_info_print_table_start>
    21ff9a:   bf 02 00 00 00          mov    $0x2,%edi
    21ff9f:   48 8d 15 4d 67 a5 00    lea    0xa5674d(%rip),%rdx        # c766f3 <arginfo_xmlwriter_void+0x233> <--- we need to receive this
    21ffa6:   31 c0                   xor    %eax,%eax
    21ffa8:   48 8d 35 bd 70 a4 00    lea    0xa470bd(%rip),%rsi        # c6706c <php_sig_gif+0x595>
    21ffaf:   e8 13 7d ff ff          call   217cc7 <php_info_print_table_row>
    21ffb4:   e8 1a 7a ff ff          call   2179d3 <php_info_print_table_end>
    21ffb9:   48 89 ef                mov    %rbp,%rdi
    21ffbc:   5d                      pop    %rbp
    21ffbd:   e9 fd 17 00 00          jmp    2217bf <display_ini_entries>

*/

TMaybe<ui64> DecodeZmInfoPhpCore(const llvm::Triple& triple, ui64 functionAddress, TConstArrayRef<ui8> bytecode) {
    auto instructionEvaluator = NPerforator::NAsm::NX86::MakeDefaultInstructionEvaluator();
    NPerforator::NAsm::NX86::TBytecodeEvaluator evaluator(
        triple,
        NPerforator::NAsm::NX86::MakeInitialState(functionAddress),
        bytecode,
        *instructionEvaluator,
        NPerforator::NAsm::NX86::MakeStopOnRetCondition());
    auto result = evaluator.Evaluate();
    if (!result.has_value()) {
        return Nothing();
    }

    auto rdxValue = GetRegisterValueOrAddress(result->State, llvm::X86::RDX);
    if (rdxValue) {
        return rdxValue;
    }

    auto edxValue = GetRegisterValueOrAddress(result->State, llvm::X86::EDX);
    if (edxValue) {
        return edxValue;
    }

    return Nothing();
}

/*

0000000000524090 <zend_vm_kind>:
  524090:   f3 0f 1e fa             endbr64
  524094:   b8 04 00 00 00          mov    $0x4,%eax <-- we need to receive this
  524099:   c3                      ret

*/

TMaybe<ui64> DecodeZendVmKind(const llvm::Triple& triple, ui64 functionAddress, TConstArrayRef<ui8> bytecode) {
    auto instructionEvaluator = NPerforator::NAsm::NX86::MakeDefaultInstructionEvaluator();
    NPerforator::NAsm::NX86::TBytecodeEvaluator evaluator(
        triple,
        NPerforator::NAsm::NX86::MakeInitialState(functionAddress),
        bytecode,
        *instructionEvaluator,
        NPerforator::NAsm::NX86::MakeStopOnRetCondition());
    auto result = evaluator.Evaluate();
    if (!result.has_value()) {
        return Nothing();
    }

    auto raxValue = GetRegisterValueOrAddress(result->State, llvm::X86::RAX);
    if (raxValue) {
        return raxValue;
    }

    auto eaxValue = GetRegisterValueOrAddress(result->State, llvm::X86::EAX);
    if (eaxValue) {
        return eaxValue;
    }

    return Nothing();
}

} // namespace NPerforator::NLinguist::NPhp::NAsm::NX86
