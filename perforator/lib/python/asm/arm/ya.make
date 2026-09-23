LIBRARY()

ADDINCL(
    ${ARCADIA_BUILD_ROOT}/contrib/libs/llvm22/lib/Target/ARM
)

PEERDIR(
    contrib/libs/llvm22/include
    contrib/libs/llvm22/lib/DebugInfo/DWARF
    contrib/libs/llvm22/lib/DebugInfo/Symbolize
    contrib/libs/llvm22/lib/MC
    contrib/libs/llvm22/lib/Object
    contrib/libs/llvm22/lib/Support
    contrib/libs/llvm22/lib/Target
    contrib/libs/llvm22/lib/Target/AArch64/Disassembler
    contrib/libs/llvm22/lib/Target/AArch64/MCTargetDesc
    contrib/libs/llvm22/lib/Target/AArch64

    perforator/lib/asm/arm
)

SRCS(
    decode.cpp
)

END()
