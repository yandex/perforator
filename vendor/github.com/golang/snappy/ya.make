GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v1.0.0)

SRCS(
    decode.go
    encode.go
    snappy.go
)

GO_TEST_SRCS(
    golden_test.go
    snappy_test.go
)

IF (ARCH_X86_64)
    SRCS(
        decode_amd64.s
        decode_asm.go
        encode_amd64.s
        encode_asm.go
    )
ENDIF()

IF (ARCH_ARM64)
    SRCS(
        decode_arm64.s
        decode_asm.go
        encode_arm64.s
        encode_asm.go
    )
ENDIF()

IF (OS_LINUX AND ARCH_ARM6 OR OS_LINUX AND ARCH_ARM7)
    SRCS(
        decode_other.go
        encode_other.go
    )
ENDIF()

END()

RECURSE(
    cmd
    gotest
)
