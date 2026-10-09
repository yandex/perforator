GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.83.2)

SRCS(
    ready_reader.go
)

GO_XTEST_SRCS(ready_reader_ext_test.go)

IF (OS_LINUX)
    SRCS(
        raw_conn_linux.go
    )
ENDIF()

IF (OS_DARWIN)
    SRCS(
        raw_conn_nonlinux.go
    )
ENDIF()

IF (OS_WINDOWS)
    SRCS(
        raw_conn_nonlinux.go
    )
ENDIF()

IF (OS_ANDROID)
    SRCS(
        raw_conn_linux.go
    )
ENDIF()

IF (OS_EMSCRIPTEN)
    SRCS(
        raw_conn_nonlinux.go
    )
ENDIF()

END()

RECURSE(
    gotest
)
