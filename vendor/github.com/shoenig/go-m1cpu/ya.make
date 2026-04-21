GO_LIBRARY(m1cpu)

LICENSE(MPL-2.0)

VERSION(v0.2.1)

GO_TEST_SRCS(examples_test.go)

IF (ARCH_X86_64)
    SRCS(
        incompatible.go
    )

    GO_TEST_SRCS(incompatible_test.go)
ENDIF()

IF (OS_LINUX AND ARCH_ARM64)
    SRCS(
        incompatible.go
    )

    GO_TEST_SRCS(incompatible_test.go)
ENDIF()

IF (OS_LINUX AND ARCH_ARM6 OR OS_LINUX AND ARCH_ARM7)
    SRCS(
        incompatible.go
    )

    GO_TEST_SRCS(incompatible_test.go)
ENDIF()

IF (OS_DARWIN AND ARCH_ARM64 AND NOT CGO_ENABLED)
    SRCS(
        incompatible.go
    )
ENDIF()

IF (OS_DARWIN AND ARCH_ARM64 AND CGO_ENABLED)
    CGO_SRCS(cpu.go)

    GO_TEST_SRCS(cpu_test.go)
ENDIF()

IF (OS_WINDOWS AND ARCH_ARM64)
    SRCS(
        incompatible.go
    )

    GO_TEST_SRCS(incompatible_test.go)
ENDIF()

IF (OS_ANDROID)
    SRCS(
        incompatible.go
    )

    GO_TEST_SRCS(incompatible_test.go)
ENDIF()

IF (OS_EMSCRIPTEN)
    SRCS(
        incompatible.go
    )

    GO_TEST_SRCS(incompatible_test.go)
ENDIF()

END()

RECURSE(
    gotest
)
