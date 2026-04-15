GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v0.51.0)

SRCS(
    ascii.go
    ciphers.go
    client_conn_pool.go
    client_priority_go126.go
    config.go
    config_go125.go
    databuffer.go
    errors.go
    flow.go
    frame.go
    gotrack.go
    http2.go
    pipe.go
    server.go
    transport.go
    unencrypted.go
    write.go
    writesched.go
    writesched_priority_rfc7540.go
    writesched_priority_rfc9218.go
    writesched_random.go
    writesched_roundrobin.go
)

END()

RECURSE(
    h2c
    hpack
)

IF (ARCH_X86_64)
    RECURSE(
        h2i
    )
ENDIF()

IF (ARCH_ARM64)
    RECURSE(
        h2i
    )
ENDIF()

IF (OS_LINUX AND ARCH_ARM6 OR OS_LINUX AND ARCH_ARM7)
    RECURSE(
        h2i
    )
ENDIF()
