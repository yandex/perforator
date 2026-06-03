GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.23.2)

SRCS(
    path.go
)

GO_TEST_SRCS(path_test.go)

IF (OS_LINUX)
    SRCS(
        path_unix.go
    )
ENDIF()

IF (OS_DARWIN)
    SRCS(
        path_unix.go
    )
ENDIF()

IF (OS_WINDOWS)
    SRCS(
        path_windows.go
    )
ENDIF()

IF (OS_ANDROID)
    SRCS(
        path_unix.go
    )
ENDIF()

IF (OS_EMSCRIPTEN)
    SRCS(
        path_unix.go
    )
ENDIF()

END()

RECURSE(
    gotest
)
