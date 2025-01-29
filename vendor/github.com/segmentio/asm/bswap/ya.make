GO_LIBRARY()

LICENSE(MIT)

VERSION(v1.2.0)

SRCS(
    swap64.go
)

GO_TEST_SRCS(swap64_test.go)

IF (ARCH_X86_64)
    SRCS(
        swap64_amd64.go
        swap64_amd64.s
    )
ENDIF()

IF (ARCH_ARM64)
    SRCS(
        swap64_default.go
    )
ENDIF()

END()

RECURSE(
    gotest
)
