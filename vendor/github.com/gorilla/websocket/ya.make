GO_LIBRARY()

LICENSE(BSD-2-Clause)

VERSION(v1.5.4-0.20250319132907-e064f32e3674)

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
    util.go
)

GO_TEST_SRCS(
    client_proxy_server_test.go
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
