LIBRARY()

ADDINCL(
    ${ARCADIA_BUILD_ROOT}/contrib/libs/llvm22/lib/Target/AArch64
)

PEERDIR(
    contrib/libs/llvm22/include
    contrib/libs/llvm22/lib/Object

    perforator/lib/elf
)

SRCS(
    decode.cpp
)

END()
