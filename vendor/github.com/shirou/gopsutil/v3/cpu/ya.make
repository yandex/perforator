GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v3.24.2)

SRCS(
    cpu.go
)

GO_TEST_SRCS(cpu_test.go)

IF (OS_LINUX)
    SRCS(
        cpu_linux.go
    )

    GO_TEST_SRCS(cpu_linux_test.go)
ENDIF()

IF (OS_DARWIN)
    SRCS(
        cpu_darwin.go
    )

    GO_TEST_SRCS(cpu_darwin_test.go)
ENDIF()

IF (OS_DARWIN AND CGO_ENABLED)
    CGO_SRCS(cpu_darwin_cgo.go)
ENDIF()

IF (OS_WINDOWS)
    SRCS(
        cpu_windows.go
    )
ENDIF()

END()

RECURSE(
    gotest
)
