GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v3.24.2)

SRCS(
    net.go
    net_linux_116.go
)

GO_TEST_SRCS(net_test.go)

IF (OS_LINUX)
    SRCS(
        net_linux.go
    )

    GO_TEST_SRCS(
        net_linux_netlink_test.go
        net_linux_test.go
    )
ENDIF()

IF (OS_DARWIN)
    SRCS(
        net_darwin.go
        net_unix.go
    )

    GO_TEST_SRCS(net_darwin_test.go)
ENDIF()

IF (OS_WINDOWS)
    SRCS(
        net_windows.go
    )
ENDIF()

END()

RECURSE(
    gotest
)
