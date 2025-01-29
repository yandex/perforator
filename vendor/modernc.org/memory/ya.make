GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v1.8.0)

SRCS(
    memory.go
    memory64.go
    nocounters.go
    trace_disabled.go
)

GO_TEST_SRCS(all_test.go)

IF (OS_LINUX)
    SRCS(
        mmap_linux_64.go
        mmap_unix.go
    )
ENDIF()

IF (OS_DARWIN)
    SRCS(
        mmap_darwin.go
        mmap_unix.go
    )
ENDIF()

IF (OS_WINDOWS)
    SRCS(
        mmap_windows.go
    )
ENDIF()

END()

RECURSE(
    gotest
)
