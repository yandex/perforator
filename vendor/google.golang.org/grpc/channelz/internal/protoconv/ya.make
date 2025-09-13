GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.73.0)

SRCS(
    channel.go
    server.go
    socket.go
    subchannel.go
    util.go
)

IF (OS_LINUX)
    SRCS(
        sockopt_linux.go
    )
ENDIF()

IF (OS_DARWIN)
    SRCS(
        sockopt_nonlinux.go
    )
ENDIF()

IF (OS_WINDOWS)
    SRCS(
        sockopt_nonlinux.go
    )
ENDIF()

END()
