GTEST()

ADDINCL(
    ${ARCADIA_BUILD_ROOT}/contrib/libs/llvm22/lib/Target/X86
)

PEERDIR(
    contrib/libs/llvm22/include
    contrib/libs/llvm22/lib/Object
    contrib/libs/llvm22/lib/Support
    contrib/libs/llvm22/lib/Target
    contrib/libs/llvm22/lib/Target/X86
    contrib/libs/llvm22/lib/Target/X86/Disassembler
    contrib/libs/llvm22/lib/Target/X86/MCTargetDesc

    library/cpp/logger/global
    library/cpp/testing/gtest
    library/cpp/testing/gtest

    perforator/lib/asm/x86
)

SRCS(
    evaluator_ut.cpp
)

END()
