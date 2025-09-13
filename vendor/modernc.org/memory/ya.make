GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v1.9.1)

SRCS(
    memory.go
    nocounters.go
    trace_disabled.go
)

GO_TEST_SRCS(all_test.go)

IF (ARCH_X86_64)
    SRCS(
        memory64.go
    )
ENDIF()

IF (ARCH_ARM64)
    SRCS(
        memory64.go
    )
ENDIF()

IF (OS_LINUX)
    SRCS(
        mmap_unix.go
    )
ENDIF()

IF (OS_LINUX AND ARCH_ARM6 OR OS_LINUX AND ARCH_ARM7)
    SRCS(
        memory32.go
    )
ENDIF()

IF (OS_DARWIN)
    SRCS(
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
