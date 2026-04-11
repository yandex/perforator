GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v4.1.25)

SRCS(
    block.go
    blocks.go
)

IF (ARCH_X86_64)
    SRCS(
        decode_amd64.s
        decode_asm.go
    )
ENDIF()

IF (ARCH_ARM64)
    SRCS(
        decode_arm64.s
        decode_asm.go
    )
ENDIF()

IF (OS_LINUX AND ARCH_ARM6 OR OS_LINUX AND ARCH_ARM7)
    SRCS(
        decode_arm.s
        decode_asm.go
    )
ENDIF()

IF (OS_EMSCRIPTEN)
    SRCS(
        decode_other.go
    )
ENDIF()

END()
