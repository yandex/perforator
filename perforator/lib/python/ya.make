LIBRARY()

INCLUDE(${ARCADIA_ROOT}/perforator/lib/arch.ya.make.inc)

IF (ARCH_x86_64)
    PEERDIR(perforator/lib/python/asm/x86)
ELSEIF (ARCH_AARCH64)
    PEERDIR(perforator/lib/python/asm/arm)
ENDIF()

PEERDIR(
    contrib/libs/llvm22/include
    contrib/libs/llvm22/lib/Object
    contrib/libs/re2

    perforator/lib/elf
    perforator/lib/tls/parser
    perforator/lib/llvmex
    perforator/lib/python/asm
)

SRCS(
    python.cpp
)

END()

RECURSE(
    asm
    cli
)

RECURSE_FOR_TESTS(
    ut
)
