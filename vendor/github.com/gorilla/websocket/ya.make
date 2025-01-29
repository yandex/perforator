GO_LIBRARY()

LICENSE(BSD-2-Clause)

VERSION(v1.5.3)

SRCS(
    client.go
    compression.go
    conn.go
    doc.go
    join.go
    json.go
    mask.go
    prepared.go
    proxy.go
    server.go
    tls_handshake.go
    util.go
    x_net_proxy.go
)

GO_TEST_SRCS(
    client_server_test.go
    client_test.go
    compression_test.go
    conn_broadcast_test.go
    conn_test.go
    join_test.go
    json_test.go
    mask_test.go
    prepared_test.go
    server_test.go
    util_test.go
)

GO_XTEST_SRCS(example_test.go)

END()

RECURSE(
    examples
    gotest
)
