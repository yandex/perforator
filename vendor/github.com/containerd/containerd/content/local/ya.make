GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.7.29)

SRCS(
    locks.go
    readerat.go
    store.go
    test_helper.go
    writer.go
)

GO_TEST_SRCS(
    locks_test.go
    store_test.go
)

IF (OS_LINUX)
    SRCS(
        store_unix.go
    )
ENDIF()

IF (OS_DARWIN)
    SRCS(
        store_bsd.go
    )
ENDIF()

IF (OS_WINDOWS)
    SRCS(
        store_windows.go
    )
ENDIF()

IF (OS_ANDROID)
    SRCS(
        store_unix.go
    )
ENDIF()

END()

RECURSE(
    gotest
)
