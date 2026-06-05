GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v2.2.0)

SRCS(
    locks.go
    readerat.go
    store.go
    writer.go
)

GO_TEST_SRCS(
    content_local_fuzz_test.go
    helper_test.go
    locks_test.go
    store_test.go
)

IF (OS_LINUX)
    SRCS(
        store_unix.go
        writer_unix.go
    )
ENDIF()

IF (OS_DARWIN)
    SRCS(
        store_bsd.go
        writer_unix.go
    )
ENDIF()

IF (OS_WINDOWS)
    SRCS(
        store_windows.go
        writer_windows.go
    )
ENDIF()

IF (OS_ANDROID)
    SRCS(
        store_unix.go
        writer_unix.go
    )
ENDIF()

IF (OS_EMSCRIPTEN)
    SRCS(
        writer_unix.go
    )
ENDIF()

END()

RECURSE(
    gotest
)
