GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.73.0)

SRCS(
    channel.go
    channelmap.go
    funcs.go
    logging.go
    server.go
    socket.go
    subchannel.go
    trace.go
)

IF (OS_LINUX)
    SRCS(
        syscall_linux.go
    )

    GO_XTEST_SRCS(syscall_test.go)
ENDIF()

IF (OS_DARWIN)
    SRCS(
        syscall_nonlinux.go
    )
ENDIF()

IF (OS_WINDOWS)
    SRCS(
        syscall_nonlinux.go
    )
ENDIF()

END()

RECURSE(
    gotest
)
