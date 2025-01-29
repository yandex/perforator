GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v3.21.11+incompatible)

SRCS(
    binary.go
    common.go
    sleep.go
)

GO_TEST_SRCS(common_test.go)

GO_XTEST_SRCS(sleep_test.go)

IF (OS_LINUX)
    SRCS(
        common_linux.go
        common_unix.go
    )
ENDIF()

IF (OS_DARWIN)
    SRCS(
        common_darwin.go
        common_unix.go
    )
ENDIF()

IF (OS_WINDOWS)
    SRCS(
        common_windows.go
    )
ENDIF()

END()

RECURSE(
    # gotest
)
