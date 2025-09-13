GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v0.42.0)

SRCS(
    ascii.go
    ciphers.go
    client_conn_pool.go
    config.go
    config_go124.go
    databuffer.go
    errors.go
    flow.go
    frame.go
    gotrack.go
    http2.go
    pipe.go
    server.go
    timer.go
    transport.go
    unencrypted.go
    write.go
    writesched.go
    writesched_priority.go
    writesched_random.go
    writesched_roundrobin.go
)

END()

RECURSE(
    h2c
    h2i
    hpack
)
