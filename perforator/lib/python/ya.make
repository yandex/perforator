LIBRARY()

ADDINCL(
    ${ARCADIA_BUILD_ROOT}/contrib/libs/llvm18/lib/Target/X86
)

PEERDIR(
    contrib/libs/llvm18/include
    contrib/libs/llvm18/lib/DebugInfo/DWARF
    contrib/libs/llvm18/lib/DebugInfo/Symbolize
    contrib/libs/llvm18/lib/MC
    contrib/libs/llvm18/lib/Object
    contrib/libs/llvm18/lib/Support
    contrib/libs/llvm18/lib/Target
    # contrib/libs/llvm18/lib/Target/AArch64/Disassembler
    # contrib/libs/llvm18/lib/Target/AArch64
    # contrib/libs/llvm18/lib/Target/ARM/Disassembler
    # contrib/libs/llvm18/lib/Target/ARM
    # contrib/libs/llvm18/lib/Target/BPF/Disassembler
    # contrib/libs/llvm18/lib/Target/BPF
    # contrib/libs/llvm18/lib/Target/LoongArch/Disassembler
    # contrib/libs/llvm18/lib/Target/LoongArch
    # contrib/libs/llvm18/lib/Target/PowerPC/Disassembler
    # contrib/libs/llvm18/lib/Target/PowerPC
    # contrib/libs/llvm18/lib/Target/WebAssembly/Disassembler
    # contrib/libs/llvm18/lib/Target/WebAssembly
    # contrib/libs/llvm18/lib/Target/NVPTX
    contrib/libs/llvm18/lib/Target/X86/Disassembler
    contrib/libs/llvm18/lib/Target/X86/MCTargetDesc
    contrib/libs/llvm18/lib/Target/X86
    contrib/libs/re2

    perforator/lib/tls/parser
    perforator/lib/llvmex
)

SRCS(
    python.cpp
)

END()

RECURSE(
    cli
)
