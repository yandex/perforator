GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.5.0)

SRCS(
    inmem_socket.go
    proxy.go
    sockets.go
    tcp_socket.go
)

GO_TEST_SRCS(inmem_socket_test.go)

IF (OS_LINUX)
    SRCS(
        sockets_unix.go
        unix_socket.go
    )

    GO_TEST_SRCS(unix_socket_test.go)
ENDIF()

IF (OS_DARWIN)
    SRCS(
        sockets_unix.go
        unix_socket.go
    )

    GO_TEST_SRCS(unix_socket_test.go)
ENDIF()

IF (OS_WINDOWS)
    SRCS(
        sockets_windows.go
    )
ENDIF()

END()

RECURSE(
    gotest
)
