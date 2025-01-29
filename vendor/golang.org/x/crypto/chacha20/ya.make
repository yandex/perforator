GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v0.31.0)

SRCS(
    chacha_generic.go
    xor.go
)

GO_TEST_SRCS(
    chacha_test.go
    vectors_test.go
)

IF (ARCH_X86_64)
    SRCS(
        chacha_noasm.go
    )
ENDIF()

IF (ARCH_ARM64)
    SRCS(
        chacha_arm64.go
        chacha_arm64.s
    )
ENDIF()

END()

RECURSE(
    gotest
)
