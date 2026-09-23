LIBRARY()

PEERDIR(
    contrib/libs/llvm22/include
    contrib/libs/llvm22/lib/Object

    perforator/lib/llvmex
)

SRCS(
    elf.cpp
)

END()
