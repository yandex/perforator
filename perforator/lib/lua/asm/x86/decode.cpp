#include "decode.h"

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

#include <util/generic/function_ref.h>
#include <util/generic/hash.h>
#include <util/generic/vector.h>

#include <perforator/lib/asm/evaluator.h>

namespace NPerforator::NLinguist::NLua::NAsm::NX86 {

/*
    Used to find offsetof(global_State, cur_L)
    Looking for setgcrefnull(g->cur_L);

    Some examples on my machine:

    0000000000015fd0 <lua_close@@Base>:
    15fd0:	41 55                	push   %r13
    15fd2:	41 bd 0a 00 00 00    	mov    $0xa,%r13d
    15fd8:	41 54                	push   %r12
    15fda:	4c 8d 25 0f fc ff ff 	lea    -0x3f1(%rip),%r12        # 15bf0 <luaL_traceback@@Base+0x7c0>
    15fe1:	55                   	push   %rbp
    15fe2:	53                   	push   %rbx
    15fe3:	48 83 ec 08          	sub    $0x8,%rsp
    15fe7:	48 8b 6f 10          	mov    0x10(%rdi),%rbp
    15feb:	48 8b 9d c0 00 00 00 	mov    0xc0(%rbp),%rbx
    15ff2:	48 89 df             	mov    %rbx,%rdi
    15ff5:	e8 96 28 ff ff       	call   8890 <luaJIT_profile_stop@plt>

    Key instructions:
    15fe7:	48 8b 6f 10          	mov    0x10(%rdi),%rbp
    16001:	48 c7 85 70 01 00 00 	movq   $0x0,0x170(%rbp)

    000000000002f470 <lua_close@@Base>:
    2f470:	55                   	push   %rbp
    2f471:	48 89 e5             	mov    %rsp,%rbp
    2f474:	41 57                	push   %r15
    2f476:	41 56                	push   %r14
    2f478:	53                   	push   %rbx
    2f479:	50                   	push   %rax
    2f47a:	4c 8b 77 10          	mov    0x10(%rdi),%r14
    2f47e:	49 8b 9e c0 00 00 00 	mov    0xc0(%r14),%rbx
    2f485:	48 89 df             	mov    %rbx,%rdi
    2f488:	ff 15 0a 79 07 00    	call   *0x7790a(%rip)        # a6d98 <luaL_pushmodule@plt+0x2b68>
    2f48e:	49 c7 86 70 01 00 00 	movq   $0x0,0x170(%r14)

    Key instructions:
    2f47a:	4c 8b 77 10          	mov    0x10(%rdi),%r14
    2f48e:	49 c7 86 70 01 00 00 	movq   $0x0,0x170(%r14)

    000000000006ba70 <lua_close@@Base>:
    6ba70:	f3 0f 1e fa          	endbr64
    6ba74:	41 55                	push   %r13
    6ba76:	41 bd 0a 00 00 00    	mov    $0xa,%r13d
    6ba7c:	41 54                	push   %r12
    6ba7e:	4c 8d 25 3b ff fe ff 	lea    -0x100c5(%rip),%r12        # 5b9c0 <lua_getinfo@@Base+0x55d0>
    6ba85:	55                   	push   %rbp
    6ba86:	53                   	push   %rbx
    6ba87:	48 83 ec 08          	sub    $0x8,%rsp
    6ba8b:	48 8b 6f 10          	mov    0x10(%rdi),%rbp
    6ba8f:	48 8b 9d c0 00 00 00 	mov    0xc0(%rbp),%rbx
    6ba96:	48 89 df             	mov    %rbx,%rdi
    6ba99:	e8 12 ce f9 ff       	call   88b0 <luaJIT_profile_stop@plt>
    6ba9e:	48 8b 73 38          	mov    0x38(%rbx),%rsi
    6baa2:	48 89 df             	mov    %rbx,%rdi
    6baa5:	48 c7 85 70 01 00 00 	movq   $0x0,0x170(%rbp)

    Key instructions:
    6ba8b:	48 8b 6f 10          	mov    0x10(%rdi),%rbp
    6baa5:	48 c7 85 70 01 00 00 	movq   $0x0,0x170(%rbp)
*/

TMaybe<i64> DecodeLuaClose(const llvm::Triple& triple, [[maybe_unused]] ui64 functionAddress, TConstArrayRef<ui8> bytecode) {
    TMaybe<i64> result;

    std::string error;
    const llvm::Target* target = llvm::TargetRegistry::lookupTarget(triple.getTriple(), error);
    if (!target) {
        return Nothing();
    }

    unsigned int globalStateRegister = 0;

    NPerforator::NAsm::DecodeInstructions(TLoggerOperator<TGlobalLog>::Log(), triple, bytecode, [&](const llvm::MCInst& inst, ui64 size) {
        Y_UNUSED(size);

        switch (inst.getOpcode()) {
            // Parse `mov disp(%rdi), reg`
            // MOV64rm [dstReg, baseReg, scaleImm, indexReg, dispImm, segReg]
            case llvm::X86::MOV64rm:
            case llvm::X86::MOV32rm: {
                auto& dst = inst.getOperand(0);
                auto& base = inst.getOperand(1);
                auto& disp = inst.getOperand(4);

                if (base.isReg() && base.getReg() == llvm::X86::RDI && dst.isReg() && disp.isImm()) {
                    globalStateRegister = dst.getReg();
                }

                break;
            }

            // Parse `movq $0, disp(reg)`
            // [baseReg, scaleImm, indexReg, dispImm, segReg, imm]
            case llvm::X86::MOV64mi32:
            case llvm::X86::MOV32mi:
            case llvm::X86::MOV16mi:
            case llvm::X86::MOV8mi: {
                auto& base = inst.getOperand(0);
                auto& disp = inst.getOperand(3);
                auto& imm = inst.getOperand(5);

                if (base.isReg() && base.getReg() == globalStateRegister && disp.isImm() && imm.isImm() && imm.getImm() == 0) {
                    // Stop the search
                    result = disp.getImm();
                    return false;
                }

                break;
            }
        }

        return true;
    });

    if (result == 0) {
        return Nothing();
    }

    return result;
}

/*
    Used to find `lj_dispatch_update` function.
    Looking for lj_dispatch_update(G(L));

    Some examples on my machine:

    000000000006e850 <luaopen_jit@@Base>:
    6e850:	41 54                	push   %r12
    6e852:	55                   	push   %rbp
    6e853:	53                   	push   %rbx
    6e854:	48 89 fb             	mov    %rdi,%rbx
    6e857:	48 83 ec 30          	sub    $0x30,%rsp
    6e85b:	48 8b 47 10          	mov    0x10(%rdi),%rax
    6e85f:	31 ff                	xor    %edi,%edi
    6e861:	48 89 e6             	mov    %rsp,%rsi
    6e864:	48 8d 68 a0          	lea    -0x60(%rax),%rbp
    6e868:	e8 2e e4 f9 ff       	call   cc9b <__cxa_finalize@plt+0x4023>
    6e86d:	85 c0                	test   %eax,%eax
    6e86f:	0f 85 1b 01 00 00    	jne    6e990 <luaopen_jit@@Base+0x140>
    6e875:	b8 01 00 ff 03       	mov    $0x3ff0001,%eax
    6e87a:	66 0f 6f 05 be 51 00 	movdqa 0x51be(%rip),%xmm0        # 73a40 <luaL_openlibs@@Base+0x1c20>
    6e881:	00
    6e882:	89 85 e8 03 00 00    	mov    %eax,0x3e8(%rbp)
    6e888:	0f 11 85 a4 09 00 00 	movups %xmm0,0x9a4(%rbp)
    6e88f:	66 0f 6f 05 b9 51 00 	movdqa 0x51b9(%rip),%xmm0        # 73a50 <luaL_openlibs@@Base+0x1c30>
    6e896:	00
    6e897:	0f 11 85 b4 09 00 00 	movups %xmm0,0x9b4(%rbp)
    6e89e:	66 0f 6f 05 ba 51 00 	movdqa 0x51ba(%rip),%xmm0        # 73a60 <luaL_openlibs@@Base+0x1c40>
    6e8a5:	00
    6e8a6:	0f 11 85 c4 09 00 00 	movups %xmm0,0x9c4(%rbp)
    6e8ad:	66 0f 6f 05 bb 51 00 	movdqa 0x51bb(%rip),%xmm0        # 73a70 <luaL_openlibs@@Base+0x1c50>
    6e8b4:	00
    6e8b5:	0f 11 85 d0 09 00 00 	movups %xmm0,0x9d0(%rbp)
    6e8bc:	48 8b 7b 10          	mov    0x10(%rbx),%rdi
    6e8c0:	e8 1b 7c fa ff       	call   164e0 <lua_close@@Base+0x510>

    Key instructions:
    6e854:	48 89 fb             	mov    %rdi,%rbx
    6e8bc:	48 8b 7b 10          	mov    0x10(%rbx),%rdi
    6e8c0:	e8 1b 7c fa ff       	call   164e0 <lua_close@@Base+0x510>

    000000000009e9f0 <luaopen_jit@@Base>:
    9e9f0:	55                   	push   %rbp
    9e9f1:	48 89 e5             	mov    %rsp,%rbp
    9e9f4:	41 57                	push   %r15
    9e9f6:	41 56                	push   %r14
    9e9f8:	53                   	push   %rbx
    9e9f9:	48 83 ec 38          	sub    $0x38,%rsp
    9e9fd:	48 89 fb             	mov    %rdi,%rbx
    9ea00:	4c 8b 77 10          	mov    0x10(%rdi),%r14
    9ea04:	45 31 ff             	xor    %r15d,%r15d
    9ea07:	48 8d 75 b0          	lea    -0x50(%rbp),%rsi
    9ea0b:	31 ff                	xor    %edi,%edi
    9ea0d:	e8 f9 48 f8 ff       	call   2330b <lua_atpanic@@Base-0x3005>
    9ea12:	85 c0                	test   %eax,%eax
    9ea14:	74 4a                	je     9ea60 <luaopen_jit@@Base+0x70>
    9ea16:	48 8d 75 c0          	lea    -0x40(%rbp),%rsi
    9ea1a:	bf 01 00 00 00       	mov    $0x1,%edi
    9ea1f:	e8 e7 48 f8 ff       	call   2330b <lua_atpanic@@Base-0x3005>
    9ea24:	85 c0                	test   %eax,%eax
    9ea26:	74 38                	je     9ea60 <luaopen_jit@@Base+0x70>
    9ea28:	44 8b 7d c8          	mov    -0x38(%rbp),%r15d
    9ea2c:	44 89 f8             	mov    %r15d,%eax
    9ea2f:	c1 e0 04             	shl    $0x4,%eax
    9ea32:	83 e0 10             	and    $0x10,%eax
    9ea35:	41 c1 ef 0e          	shr    $0xe,%r15d
    9ea39:	41 83 e7 20          	and    $0x20,%r15d
    9ea3d:	41 09 c7             	or     %eax,%r15d
    9ea40:	83 7d b0 07          	cmpl   $0x7,-0x50(%rbp)
    9ea44:	72 1a                	jb     9ea60 <luaopen_jit@@Base+0x70>
    9ea46:	48 8d 75 d0          	lea    -0x30(%rbp),%rsi
    9ea4a:	bf 07 00 00 00       	mov    $0x7,%edi
    9ea4f:	e8 b7 48 f8 ff       	call   2330b <lua_atpanic@@Base-0x3005>
    9ea54:	8b 45 d4             	mov    -0x2c(%rbp),%eax
    9ea57:	c1 e8 02             	shr    $0x2,%eax
    9ea5a:	83 e0 40             	and    $0x40,%eax
    9ea5d:	41 09 c7             	or     %eax,%r15d
    9ea60:	41 81 cf 01 00 ff 03 	or     $0x3ff0001,%r15d
    9ea67:	45 89 be 88 03 00 00 	mov    %r15d,0x388(%r14)
    9ea6e:	c5 fc 10 05 5a 0b f7 	vmovups -0x8f4a6(%rip),%ymm0        # f5d0 <lua_atpanic@@Base-0x16d40>
    9ea75:	ff
    9ea76:	c4 c1 7c 11 86 44 09 	vmovups %ymm0,0x944(%r14)
    9ea7d:	00 00
    9ea7f:	c5 fc 10 05 65 0b f7 	vmovups -0x8f49b(%rip),%ymm0        # f5ec <lua_atpanic@@Base-0x16d24>
    9ea86:	ff
    9ea87:	c4 c1 7c 11 86 60 09 	vmovups %ymm0,0x960(%r14)
    9ea8e:	00 00
    9ea90:	48 8b 7b 10          	mov    0x10(%rbx),%rdi
    9ea94:	c5 f8 77             	vzeroupper
    9ea97:	e8 b4 14 f9 ff       	call   2ff50 <lua_close@@Base+0xae0>

    Key instructions:
    9e9fd:	48 89 fb             	mov    %rdi,%rbx
    9ea90:	48 8b 7b 10          	mov    0x10(%rbx),%rdi
    9ea97:	e8 b4 14 f9 ff       	call   2ff50 <lua_close@@Base+0xae0>

    0000000000076580 <luaopen_jit@@Base>:
    76580:	f3 0f 1e fa          	endbr64
    76584:	41 54                	push   %r12
    76586:	55                   	push   %rbp
    76587:	53                   	push   %rbx
    76588:	48 89 fb             	mov    %rdi,%rbx
    7658b:	48 83 ec 30          	sub    $0x30,%rsp
    7658f:	48 8b 47 10          	mov    0x10(%rdi),%rax
    76593:	31 ff                	xor    %edi,%edi
    76595:	48 89 e6             	mov    %rsp,%rsi
    76598:	48 8d 68 90          	lea    -0x70(%rax),%rbp
    7659c:	e8 ca 67 f9 ff       	call   cd6b <__cxa_finalize@plt+0x40e3>
    765a1:	85 c0                	test   %eax,%eax
    765a3:	0f 85 a7 01 00 00    	jne    76750 <luaopen_jit@@Base+0x1d0>
    765a9:	b8 01 00 ff 03       	mov    $0x3ff0001,%eax
    765ae:	66 0f 6f 05 8a 8a 00 	movdqa 0x8a8a(%rip),%xmm0        # 7f040 <str_hash_init_sse42@@Base+0x6ac0>
    765b5:	00
    765b6:	89 85 00 04 00 00    	mov    %eax,0x400(%rbp)
    765bc:	0f 11 85 bc 09 00 00 	movups %xmm0,0x9bc(%rbp)
    765c3:	66 0f 6f 05 85 8a 00 	movdqa 0x8a85(%rip),%xmm0        # 7f050 <str_hash_init_sse42@@Base+0x6ad0>
    765ca:	00
    765cb:	0f 11 85 cc 09 00 00 	movups %xmm0,0x9cc(%rbp)
    765d2:	66 0f 6f 05 86 8a 00 	movdqa 0x8a86(%rip),%xmm0        # 7f060 <str_hash_init_sse42@@Base+0x6ae0>
    765d9:	00
    765da:	0f 11 85 dc 09 00 00 	movups %xmm0,0x9dc(%rbp)
    765e1:	66 0f 6f 05 87 8a 00 	movdqa 0x8a87(%rip),%xmm0        # 7f070 <str_hash_init_sse42@@Base+0x6af0>
    765e8:	00
    765e9:	0f 11 85 e8 09 00 00 	movups %xmm0,0x9e8(%rbp)
    765f0:	48 8b 7b 10          	mov    0x10(%rbx),%rdi
    765f4:	48 8d 2d 90 80 00 00 	lea    0x8090(%rip),%rbp        # 7e68b <str_hash_init_sse42@@Base+0x610b>
    765fb:	e8 e0 f0 f9 ff       	call   156e0 <__cxa_finalize@plt+0xca58>

    Key instructions:
    76588:	48 89 fb             	mov    %rdi,%rbx
    765f0:	48 8b 7b 10          	mov    0x10(%rbx),%rdi
    765fb:	e8 e0 f0 f9 ff       	call   156e0 <__cxa_finalize@plt+0xca58>
*/

TMaybe<i64> DecodeLuaOpenJit(const llvm::Triple& triple, [[maybe_unused]] ui64 functionAddress, TConstArrayRef<ui8> bytecode) {
    TMaybe<i64> result;

    std::string error;
    const llvm::Target* target = llvm::TargetRegistry::lookupTarget(triple.getTriple(), error);
    if (!target) {
        return Nothing();
    }

    i64 pc = 0;
    unsigned int luaStateRegister = 0;
    bool globalStateSeen = false;

    NPerforator::NAsm::DecodeInstructions(TLoggerOperator<TGlobalLog>::Log(), triple, bytecode, [&](const llvm::MCInst& inst, ui64 size) {
        pc += size;

        switch (inst.getOpcode()) {
            // Parse `mov reg, rdi`
            case llvm::X86::MOV64rr:
            case llvm::X86::MOV32rr: {
                auto& dst = inst.getOperand(0);
                auto& src = inst.getOperand(1);

                if (dst.isReg() && src.isReg() &&
                    src.getReg() == llvm::X86::RDI) {
                    luaStateRegister = dst.getReg();
                }

                break;
            }

            // Parse `0x10(reg), %rdi`
            case llvm::X86::MOV64rm:
            case llvm::X86::MOV32rm: {
                auto& dst = inst.getOperand(0);
                auto& base = inst.getOperand(1);
                auto& disp = inst.getOperand(4);

                if (dst.isReg() && dst.getReg() == llvm::X86::RDI && base.isReg() && base.getReg() == luaStateRegister && disp.isImm() && disp.getImm() == 0x10) {
                    globalStateSeen = true;
                }

                break;
            }

            // Parse `call rel`
            case llvm::X86::CALL64pcrel32: {
                auto& rel = inst.getOperand(0);

                if (rel.isImm() && globalStateSeen) {
                    // Stop the search
                    auto callAddr = pc + rel.getImm();
                    result = callAddr;
                    return false;
                }

                break;
            }
        }

        return true;
    });

    return result;
}

/*
    Used to find `GG_G2DISP`.
    Looking for `G2GG(g)->dispatch;`

    Some examples on my machine:

    164e0:       0f b6 97 88 03 00 00    movzbl 0x388(%rdi),%edx
    164e7:       8b 87 cc 03 00 00       mov    0x3cc(%rdi),%eax
    164ed:       49 89 f8                mov    %rdi,%r8
    164f0:       be 25 00 00 00          mov    $0x25,%esi
    164f5:       44 0f b6 9f 92 00 00    movzbl 0x92(%rdi),%r11d
    164fc:       00
    164fd:       c1 e2 04                shl    $0x4,%edx
    16500:       83 e2 10                and    $0x10,%edx
    16503:       85 c0                   test   %eax,%eax
    16505:       75 02                   jne    16509 <lua_close@@Base+0x539>
    16507:       31 f6                   xor    %esi,%esi
    16509:       41 0f b6 80 91 00 00    movzbl 0x91(%r8),%eax
    16510:       00
    16511:       89 c1                   mov    %eax,%ecx
    16513:       c0 f9 07                sar    $0x7,%cl
    16516:       83 e1 44                and    $0x44,%ecx
    16519:       a8 0c                   test   $0xc,%al
    1651b:       41 0f 95 c1             setne  %r9b
    1651f:       83 e0 03                and    $0x3,%eax
    16522:       41 c1 e1 02             shl    $0x2,%r9d
    16526:       41 09 d1                or     %edx,%r9d
    16529:       41 09 f1                or     %esi,%r9d
    1652c:       41 09 c9                or     %ecx,%r9d
    1652f:       41 09 c1                or     %eax,%r9d
    16532:       45 38 cb                cmp    %r9b,%r11b
    16535:       0f 84 b5 01 00 00       je     166f0 <lua_close@@Base+0x720>
    1653b:       44 89 c8                mov    %r9d,%eax
    1653e:       41 54                   push   %r12
    16540:       4c 8d 15 29 28 ff ff    lea    -0xd7d7(%rip),%r10        # 8d70 <__cxa_finalize@plt+0xf8>
    16547:       83 e0 30                and    $0x30,%eax
    1654a:       55                      push   %rbp
    1654b:       53                      push   %rbx
    1654c:       45 88 88 92 00 00 00    mov    %r9b,0x92(%r8)
    16553:       3c 10                   cmp    $0x10,%al
    16555:       0f 84 f5 00 00 00       je     16650 <lua_close@@Base+0x680>
    1655b:       0f b7 1d 32 d8 05 00    movzwl 0x5d832(%rip),%ebx        # 73d94 <luaL_openlibs@@Base+0x1f74>
    16562:       0f b7 2d 31 d8 05 00    movzwl 0x5d831(%rip),%ebp        # 73d9a <luaL_openlibs@@Base+0x1f7a>
    16569:       48 8d 05 70 3d ff ff    lea    -0xc290(%rip),%rax        # a2e0 <__cxa_finalize@plt+0x1668>
    16570:       49 8b b8 d8 16 00 00    mov    0x16d8(%r8),%rdi
    16577:       49 8b b0 f0 16 00 00    mov    0x16f0(%r8),%rsi
    1657e:       49 8b 88 08 17 00 00    mov    0x1708(%r8),%rcx
    16585:       4c 01 d3                add    %r10,%rbx
    16588:       4c 01 d5                add    %r10,%rbp
    1658b:       45 89 dc                mov    %r11d,%r12d
    1658e:       44 89 ca                mov    %r9d,%edx
    16591:       49 89 b8 d0 16 00 00    mov    %rdi,0x16d0(%r8)
    16598:       45 31 cc                xor    %r9d,%r12d
    1659b:       49 89 b0 e8 16 00 00    mov    %rsi,0x16e8(%r8)
    165a2:       83 e2 04                and    $0x4,%edx
    165a5:       49 89 80 88 16 00 00    mov    %rax,0x1688(%r8)
    165ac:       49 89 88 00 17 00 00    mov    %rcx,0x1700(%r8)
    165b3:       41 f6 c4 64             test   $0x64,%r12b
    165b7:       0f 84 3b 01 00 00       je     166f8 <lua_close@@Base+0x728>
    165bd:       49 8d 80 88 0f 00 00    lea    0xf88(%r8),%rax

    Key instructions:
    164ed:       49 89 f8                mov    %rdi,%r8
    165bd:       49 8d 80 88 0f 00 00    lea    0xf88(%r8),%rax

    2ff50:       55                      push   %rbp
    2ff51:       48 89 e5                mov    %rsp,%rbp
    2ff54:       41 57                   push   %r15
    2ff56:       41 56                   push   %r14
    2ff58:       41 55                   push   %r13
    2ff5a:       41 54                   push   %r12
    2ff5c:       53                      push   %rbx
    2ff5d:       48 83 ec 18             sub    $0x18,%rsp
    2ff61:       8b 9f 88 03 00 00       mov    0x388(%rdi),%ebx
    2ff67:       8b 87 cc 03 00 00       mov    0x3cc(%rdi),%eax
    2ff6d:       c1 e3 04                shl    $0x4,%ebx
    2ff70:       83 e3 10                and    $0x10,%ebx
    2ff73:       85 c0                   test   %eax,%eax
    2ff75:       ba 25 00 00 00          mov    $0x25,%edx
    2ff7a:       0f 44 d0                cmove  %eax,%edx
    2ff7d:       09 da                   or     %ebx,%edx
    2ff7f:       44 0f b6 a7 91 00 00    movzbl 0x91(%rdi),%r12d
    2ff86:       00
    2ff87:       31 f6                   xor    %esi,%esi
    2ff89:       41 f6 c4 0c             test   $0xc,%r12b
    2ff8d:       40 0f 95 c6             setne  %sil
    2ff91:       c1 e6 02                shl    $0x2,%esi
    2ff94:       45 89 e7                mov    %r12d,%r15d
    2ff97:       41 83 e7 03             and    $0x3,%r15d
    2ff9b:       45 84 e4                test   %r12b,%r12b
    2ff9e:       b9 44 00 00 00          mov    $0x44,%ecx
    2ffa3:       0f 49 ce                cmovns %esi,%ecx
    2ffa6:       44 0f b6 b7 92 00 00    movzbl 0x92(%rdi),%r14d
    2ffad:       00
    2ffae:       09 d1                   or     %edx,%ecx
    2ffb0:       41 09 cf                or     %ecx,%r15d
    2ffb3:       45 39 f7                cmp    %r14d,%r15d
    2ffb6:       0f 84 13 05 00 00       je     304cf <lua_close@@Base+0x105f>
    2ffbc:       44 88 bf 92 00 00 00    mov    %r15b,0x92(%rdi)
    2ffc3:       83 e2 30                and    $0x30,%edx
    2ffc6:       4c 8d 15 13 f4 fe ff    lea    -0x10bed(%rip),%r10        # 1f3e0 <lua_atpanic@@Base-0x6f30>
    2ffcd:       83 fa 10                cmp    $0x10,%edx
    2ffd0:       75 3a                   jne    3000c <lua_close@@Base+0xb9c>
    2ffd2:       0f b7 15 a5 9b fd ff    movzwl -0x2645b(%rip),%edx        # 9b7e <lua_atpanic@@Base-0x1c792>
    2ffd9:       4c 01 d2                add    %r10,%rdx
    2ffdc:       0f b7 35 a1 9b fd ff    movzwl -0x2645f(%rip),%esi        # 9b84 <lua_atpanic@@Base-0x1c78c>
    2ffe3:       4c 01 d6                add    %r10,%rsi
    2ffe6:       44 0f b7 05 7e 9b fd    movzwl -0x26482(%rip),%r8d        # 9b6c <lua_atpanic@@Base-0x1c7a4>
    2ffed:       ff
    2ffee:       4d 01 d0                add    %r10,%r8
    2fff1:       44 0f b7 0d 91 9b fd    movzwl -0x2646f(%rip),%r9d        # 9b8a <lua_atpanic@@Base-0x1c786>
    2fff8:       ff
    2fff9:       4d 01 d1                add    %r10,%r9
    2fffc:       4c 8d 1d 95 9b fd ff    lea    -0x2646b(%rip),%r11        # 9b98 <lua_atpanic@@Base-0x1c778>
    30003:       4c 8d 15 88 9b fd ff    lea    -0x26478(%rip),%r10        # 9b92 <lua_atpanic@@Base-0x1c77e>
    3000a:       eb 2a                   jmp    30036 <lua_close@@Base+0xbc6>
    3000c:       48 8b 97 d8 16 00 00    mov    0x16d8(%rdi),%rdx
    30013:       48 8b b7 f0 16 00 00    mov    0x16f0(%rdi),%rsi
    3001a:       4c 8b 8f 08 17 00 00    mov    0x1708(%rdi),%r9
    30021:       4c 8d 1d 72 9b fd ff    lea    -0x2648e(%rip),%r11        # 9b9a <lua_atpanic@@Base-0x1c776>
    30028:       4c 8d 15 65 9b fd ff    lea    -0x2649b(%rip),%r10        # 9b94 <lua_atpanic@@Base-0x1c77c>
    3002f:       4c 8d 05 1a 09 ff ff    lea    -0xf6e6(%rip),%r8        # 20950 <lua_atpanic@@Base-0x59c0>
    30036:       45 0f b7 1b             movzwl (%r11),%r11d
    3003a:       45 0f b7 12             movzwl (%r10),%r10d
    3003e:       4c 89 55 c0             mov    %r10,-0x40(%rbp)
    30042:       48 89 97 d0 16 00 00    mov    %rdx,0x16d0(%rdi)
    30049:       48 89 b7 e8 16 00 00    mov    %rsi,0x16e8(%rdi)
    30050:       4c 89 87 88 16 00 00    mov    %r8,0x1688(%rdi)
    30057:       4c 89 8f 00 17 00 00    mov    %r9,0x1700(%rdi)
    3005e:       45 89 fd                mov    %r15d,%r13d
    30061:       45 31 f5                xor    %r14d,%r13d
    30064:       41 f6 c5 64             test   $0x64,%r13b
    30068:       74 4b                   je     300b5 <lua_close@@Base+0xc45>
    3006a:       f6 c1 04                test   $0x4,%cl
    3006d:       0f 85 a2 00 00 00       jne    30115 <lua_close@@Base+0xca5>
    30073:       48 8d 87 88 0f 00 00    lea    0xf88(%rdi),%rax

    Key instructions:
    30073:       48 8d 87 88 0f 00 00    lea    0xf88(%rdi),%rax
    no RDI copying!

    156e0:       0f b6 8f 90 03 00 00    movzbl 0x390(%rdi),%ecx
    156e7:       8b 87 d4 03 00 00       mov    0x3d4(%rdi),%eax
    156ed:       48 89 fa                mov    %rdi,%rdx
    156f0:       44 0f b6 97 92 00 00    movzbl 0x92(%rdi),%r10d
    156f7:       00
    156f8:       bf 25 00 00 00          mov    $0x25,%edi
    156fd:       c1 e1 04                shl    $0x4,%ecx
    15700:       83 e1 10                and    $0x10,%ecx
    15703:       85 c0                   test   %eax,%eax
    15705:       75 02                   jne    15709 <__cxa_finalize@plt+0xca81>
    15707:       31 ff                   xor    %edi,%edi
    15709:       0f b6 82 91 00 00 00    movzbl 0x91(%rdx),%eax
    15710:       89 c6                   mov    %eax,%esi
    15712:       40 c0 fe 07             sar    $0x7,%sil
    15716:       83 e6 44                and    $0x44,%esi
    15719:       a8 0c                   test   $0xc,%al
    1571b:       41 0f 95 c0             setne  %r8b
    1571f:       83 e0 03                and    $0x3,%eax
    15722:       41 c1 e0 02             shl    $0x2,%r8d
    15726:       41 09 c8                or     %ecx,%r8d
    15729:       41 09 f8                or     %edi,%r8d
    1572c:       41 09 f0                or     %esi,%r8d
    1572f:       41 09 c0                or     %eax,%r8d
    15732:       45 38 c2                cmp    %r8b,%r10b
    15735:       0f 84 7d 01 00 00       je     158b8 <__cxa_finalize@plt+0xcc30>
    1573b:       44 89 c0                mov    %r8d,%eax
    1573e:       55                      push   %rbp
    1573f:       4c 8d 0d fa 36 ff ff    lea    -0xc906(%rip),%r9        # 8e40 <__cxa_finalize@plt+0x1b8>
    15746:       83 e0 30                and    $0x30,%eax
    15749:       53                      push   %rbx
    1574a:       44 88 82 92 00 00 00    mov    %r8b,0x92(%rdx)
    15751:       3c 10                   cmp    $0x10,%al
    15753:       0f 84 df 00 00 00       je     15838 <__cxa_finalize@plt+0xcbb0>
    15759:       48 8b ba 00 17 00 00    mov    0x1700(%rdx),%rdi
    15760:       48 8b b2 18 17 00 00    mov    0x1718(%rdx),%rsi
    15767:       4c 8d 1d 3e 52 ff ff    lea    -0xadc2(%rip),%r11        # a9ac <__cxa_finalize@plt+0x1d24>
    1576e:       48 8d 0d 3b 4c ff ff    lea    -0xb3c5(%rip),%rcx        # a3b0 <__cxa_finalize@plt+0x1728>
    15775:       48 8b 82 30 17 00 00    mov    0x1730(%rdx),%rax
    1577c:       44 89 d3                mov    %r10d,%ebx
    1577f:       44 89 c5                mov    %r8d,%ebp
    15782:       48 89 ba f8 16 00 00    mov    %rdi,0x16f8(%rdx)
    15789:       44 31 c3                xor    %r8d,%ebx
    1578c:       48 89 b2 10 17 00 00    mov    %rsi,0x1710(%rdx)
    15793:       83 e5 04                and    $0x4,%ebp
    15796:       48 89 8a b0 16 00 00    mov    %rcx,0x16b0(%rdx)
    1579d:       48 89 82 28 17 00 00    mov    %rax,0x1728(%rdx)
    157a4:       f6 c3 64                test   $0x64,%bl
    157a7:       0f 84 13 01 00 00       je     158c0 <__cxa_finalize@plt+0xcc38>
    157ad:       48 8d 82 b0 0f 00 00    lea    0xfb0(%rdx),%rax

    Key instructions:
    157ad:       48 8d 82 b0 0f 00 00    lea    0xfb0(%rdx),%rax
*/

TMaybe<i64> DecodeLjDispatchUpdate(const llvm::Triple& triple, [[maybe_unused]] ui64 functionAddress, TConstArrayRef<ui8> bytecode) {
    TMaybe<i64> result;

    std::string error;
    const llvm::Target* target = llvm::TargetRegistry::lookupTarget(triple.getTriple(), error);
    if (!target) {
        return Nothing();
    }

    unsigned int globalStateRegister = 0;

    NPerforator::NAsm::DecodeInstructions(TLoggerOperator<TGlobalLog>::Log(), triple, bytecode, [&](const llvm::MCInst& inst, ui64 size) {
        Y_UNUSED(size);

        switch (inst.getOpcode()) {
            // mov
            case llvm::X86::MOV64rr:
            case llvm::X86::MOV32rr: {
                const auto& dst = inst.getOperand(0);
                const auto& src = inst.getOperand(1);

                if (dst.isReg() && src.isReg() && src.getReg() == llvm::X86::RDI) {
                    globalStateRegister = dst.getReg();
                }

                break;
            }

            // lea disp(reg),reg
            case llvm::X86::LEA64r:
            case llvm::X86::LEA32r: {
                const auto& base = inst.getOperand(1);
                const auto& disp = inst.getOperand(4);

                if (base.isReg() && base.getReg() == globalStateRegister && disp.isImm()) {
                    // Stop the search
                    result = disp.getImm();
                    return false;
                } else if (base.isReg() && base.getReg() == llvm::X86::RDI && disp.isImm()) {
                    // Another case - RDI used directly (no copying)

                    // Stop the search
                    result = disp.getImm();
                    return false;
                }

                break;
            }
        }

        return true;
    });

    return result;
}

/*
    Used to find address of `lj_gc_step`.
    Unlike `lj_gc_fullgc`, `lj_gc_step` doesn't inline in my examples.

    Looking for `(GCSize)data << 10` then `lj_gc_step(L)`.

    48 c1 e2 0a          	shl    rdx,0xa
    48 89 c8             	mov    rax,rcx
    48 29 d0             	sub    rax,rdx
    48 39 d1             	cmp    rcx,rdx
    ba 00 00 00 00       	mov    edx,0x0
    48 0f 42 c2          	cmovb  rax,rdx
    48 89 43 18          	mov    QWORD PTR [rbx+0x18],rax
    eb 19                	jmp    6590c <lua_gc+0x19c>
    0f 1f 44 00 00       	nop    DWORD PTR [rax+rax*1+0x0]
    48 89 ef             	mov    rdi,rbp
    e8 e0 89 fd ff       	call   3e2e0 <luaL_where+0xe6a0>

    Key instructions:
    shl    rdx,0xa                   // Wait for this shift
    call   3e2e0 <luaL_where+0xe6a0> // Find following `call` instruction
*/

TMaybe<i64> DecodeLuaGc(const llvm::Triple& triple, [[maybe_unused]] ui64 functionAddress, TConstArrayRef<ui8> bytecode) {
    static constexpr i64 kShift = 0xA;

    TMaybe<i64> result;

    std::string error;
    const llvm::Target* target = llvm::TargetRegistry::lookupTarget(triple.getTriple(), error);
    if (!target) {
        return Nothing();
    }

    i64 pc = 0;
    bool shiftSeen = false;

    NPerforator::NAsm::DecodeInstructions(TLoggerOperator<TGlobalLog>::Log(), triple, bytecode, [&](const llvm::MCInst& inst, ui64 size) {
        pc += size;

        switch (inst.getOpcode()) {
            // `shl reg,0xa`
            case llvm::X86::SHL64ri: {
                auto& imm = inst.getOperand(2);

                if (imm.isImm() && imm.getImm() == kShift) {
                    shiftSeen = true;
                }

                break;
            }

            // `call rel`
            case llvm::X86::CALL64pcrel32: {
                auto& rel = inst.getOperand(0);

                if (rel.isImm() && shiftSeen) {
                    // Stop the search
                    auto callAddress = pc + rel.getImm();
                    result = callAddress;
                    return false;
                }

                break;
            }
        }

        return true;
    });

    return result;
}

/*
    Used to find `offsetof(global_State, vmstate)`
    Looking for write of negative int value.
    Some LuaJIT implementations have different VM states, to compensate this, we will look for any negative value we found first.

    000000000000e450 <lj_gc_step>:
    e450:	f3 0f 1e fa          	endbr64
    e454:	41 55                	push   r13
    e456:	41 54                	push   r12
    e458:	49 89 fc             	mov    r12,rdi
    e45b:	55                   	push   rbp
    e45c:	53                   	push   rbx
    e45d:	48 83 ec 08          	sub    rsp,0x8
    e461:	48 8b 6f 10          	mov    rbp,QWORD PTR [rdi+0x10]
    e465:	8b 45 68             	mov    eax,DWORD PTR [rbp+0x68]
    e468:	48 8b 55 18          	mov    rdx,QWORD PTR [rbp+0x18]
    e46c:	44 8b ad b8 00 00 00 	mov    r13d,DWORD PTR [rbp+0xb8]
    e473:	c7 85 b8 00 00 00 fd 	mov    DWORD PTR [rbp+0xb8],0xfffffffd

    Key instruction:
    e473:	c7 85 b8 00 00 00 fd 	mov    DWORD PTR [rbp+0xb8],0xfffffffd
*/

TMaybe<i64> DecodeLjGcStep(const llvm::Triple& triple, [[maybe_unused]] ui64 functionAddress, TConstArrayRef<ui8> bytecode) {
    TMaybe<i64> result;

    std::string error;
    const llvm::Target* target = llvm::TargetRegistry::lookupTarget(triple.getTriple(), error);
    if (!target) {
        return Nothing();
    }

    NPerforator::NAsm::DecodeInstructions(TLoggerOperator<TGlobalLog>::Log(), triple, bytecode, [&](const llvm::MCInst& inst, ui64 size) {
        Y_UNUSED(size);

        switch (inst.getOpcode()) {
            // `mov reg+disp,imm`
            case llvm::X86::MOV32mi: {
                auto& base = inst.getOperand(0);
                auto& disp = inst.getOperand(3);
                auto& imm = inst.getOperand(5);

                if (base.isReg() && disp.isImm() && imm.isImm() && static_cast<int32_t>(imm.getImm()) < 0) {
                    // Stop the search
                    result = disp.getImm();
                    return false;
                }

                break;
            }
        }

        return true;
    });

    if (result == 0) {
        return Nothing();
    }

    return result;
}

} // namespace NPerforator::NLinguist::NLua::NAsm::NX86
