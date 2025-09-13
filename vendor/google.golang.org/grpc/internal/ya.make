GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.73.0)

SRCS(
    experimental.go
    internal.go
)

IF (OS_LINUX)
    SRCS(
        tcp_keepalive_unix.go
    )
ENDIF()

IF (OS_DARWIN)
    SRCS(
        tcp_keepalive_unix.go
    )
ENDIF()

IF (OS_WINDOWS)
    SRCS(
        tcp_keepalive_windows.go
    )
ENDIF()

END()

RECURSE(
    admin
    backoff
    balancer
    balancergroup
    balancerload
    binarylog
    buffer
    cache
    channelz
    credentials
    envconfig
    googlecloud
    grpclog
    grpcsync
    grpctest
    grpcutil
    hierarchy
    idle
    leakcheck
    metadata
    pretty
    profiling
    proto
    proxyattributes
    resolver
    ringhash
    serviceconfig
    stats
    status
    stubserver
    testutils
    transport
    wrr
    xds
    # yo
)

IF (OS_LINUX)
    RECURSE(
        syscall
    )
ENDIF()

IF (OS_DARWIN)
    RECURSE(
        syscall
    )
ENDIF()

IF (OS_WINDOWS)
    RECURSE(
        syscall
    )
ENDIF()
