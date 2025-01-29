GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v4.1.21)

SRCS(
    block.go
    blocks.go
    decode_asm.go
)

IF (ARCH_X86_64)
    SRCS(
        decode_amd64.s
    )
ENDIF()

IF (ARCH_ARM64)
    SRCS(
        decode_arm64.s
    )
ENDIF()

END()
