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
    contrib/libs/llvm18/lib/Target/X86/Disassembler
    contrib/libs/llvm18/lib/Target/X86/MCTargetDesc
    contrib/libs/llvm18/lib/Target/X86
    perforator/lib/asm/x86
)

SRCS(
    decode.cpp
)

END()
