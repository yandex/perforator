GO_LIBRARY()

LICENSE(MIT-0)

VERSION(v1.2.1)

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

IF (OS_LINUX AND ARCH_ARM6 OR OS_LINUX AND ARCH_ARM7)
    SRCS(
        swap64_default.go
    )
ENDIF()

IF (OS_EMSCRIPTEN)
    SRCS(
        swap64_default.go
    )
ENDIF()

END()

RECURSE(
    gotest
)
