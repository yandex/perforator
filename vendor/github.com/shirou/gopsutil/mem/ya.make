GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v3.21.11+incompatible)

SRCS(
    mem.go
)

GO_TEST_SRCS(mem_test.go)

IF (OS_LINUX)
    SRCS(
        mem_linux.go
    )

    GO_TEST_SRCS(mem_linux_test.go)
ENDIF()

IF (OS_DARWIN)
    SRCS(
        mem_darwin.go
    )

    GO_TEST_SRCS(mem_darwin_test.go)
ENDIF()

IF (OS_DARWIN AND CGO_ENABLED)
    CGO_SRCS(mem_darwin_cgo.go)
ENDIF()

IF (OS_WINDOWS)
    SRCS(
        mem_windows.go
    )
ENDIF()

END()

RECURSE(
    # gotest
)
