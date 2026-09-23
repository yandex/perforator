LIBRARY()

IF (ARCH_X86_64)
    PEERDIR(
        perforator/lib/pthread/asm/x86
    )
ELSEIF (ARCH_AARCH64)
    PEERDIR(
        perforator/lib/pthread/asm/arm
    )
ENDIF()

PEERDIR(
    contrib/libs/llvm22/include
    contrib/libs/llvm22/lib/Object

    perforator/lib/elf
    perforator/lib/llvmex
)

SRCS(
    pthread.cpp
)

END()

RECURSE(
    asm
    cli
)

IF (NOT OPENSOURCE)
    RECURSE_FOR_TESTS(ut)
ENDIF()
