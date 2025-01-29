GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v0.31.0)

SRCS(
    poly1305.go
    sum_generic.go
)

GO_TEST_SRCS(
    poly1305_test.go
    vectors_test.go
)

IF (ARCH_X86_64)
    SRCS(
        sum_amd64.go
        sum_amd64.s
    )
ENDIF()

IF (ARCH_ARM64)
    SRCS(
        mac_noasm.go
    )
ENDIF()

END()

RECURSE(
    gotest
)
