PROGRAM()

INCLUDE(${ARCADIA_ROOT}/perforator/lib/arch.ya.make.inc)

IF (ARCH_X86)
    PEERDIR(
        contrib/libs/llvm22/lib/Target/X86/Disassembler
    )
ELSEIF (ARCH_AARCH64)
    PEERDIR(
        contrib/libs/llvm22/lib/Target/AArch64/Disassembler
    )
ENDIF()

PEERDIR(
    contrib/libs/llvm22/include
    contrib/libs/llvm22/lib/Object

    library/cpp/logger/global

    perforator/lib/pthread
)

SRCS(
    main.cpp
)

END()
