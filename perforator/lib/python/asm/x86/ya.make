LIBRARY()

ADDINCL(
    ${ARCADIA_BUILD_ROOT}/contrib/libs/llvm22/lib/Target/X86
)

PEERDIR(
    contrib/libs/llvm22/include
    contrib/libs/llvm22/lib/DebugInfo/DWARF
    contrib/libs/llvm22/lib/DebugInfo/Symbolize
    contrib/libs/llvm22/lib/MC
    contrib/libs/llvm22/lib/Object
    contrib/libs/llvm22/lib/Support
    contrib/libs/llvm22/lib/Target
    contrib/libs/llvm22/lib/Target/X86/Disassembler
    contrib/libs/llvm22/lib/Target/X86/MCTargetDesc
    contrib/libs/llvm22/lib/Target/X86

    perforator/lib/asm/x86
)

SRCS(
    decode.cpp
)

END()
