GTEST()

PEERDIR(
    contrib/libs/llvm22/include
    contrib/libs/llvm22/lib/Object

    perforator/lib/elf
    perforator/lib/llvmex
)

SRCS(
    elf_ut.cpp
)

END()
