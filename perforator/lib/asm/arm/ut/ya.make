GTEST()

ADDINCL(
    ${ARCADIA_BUILD_ROOT}/contrib/libs/llvm22/lib/Target/ARM
)

PEERDIR(
    contrib/libs/llvm22/include
    contrib/libs/llvm22/lib/Object
    contrib/libs/llvm22/lib/Support
    contrib/libs/llvm22/lib/Target
    contrib/libs/llvm22/lib/Target/ARM
    contrib/libs/llvm22/lib/Target/ARM/Disassembler
    contrib/libs/llvm22/lib/Target/ARM/MCTargetDesc

    library/cpp/logger/global
    library/cpp/testing/gtest
    library/cpp/testing/gtest

    perforator/lib/asm/arm
)

SRCS(
    evaluator_ut.cpp
)

END()
