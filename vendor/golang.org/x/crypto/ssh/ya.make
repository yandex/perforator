GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v0.31.0)

SRCS(
    buffer.go
    certs.go
    channel.go
    cipher.go
    client.go
    client_auth.go
    common.go
    connection.go
    doc.go
    handshake.go
    kex.go
    keys.go
    mac.go
    messages.go
    mux.go
    server.go
    session.go
    ssh_gss.go
    streamlocal.go
    tcpip.go
    transport.go
)

GO_TEST_SRCS(
    benchmark_test.go
    buffer_test.go
    certs_test.go
    cipher_test.go
    client_auth_test.go
    client_test.go
    common_test.go
    handshake_test.go
    kex_test.go
    keys_test.go
    mempipe_test.go
    messages_test.go
    mux_test.go
    server_multi_auth_test.go
    server_test.go
    session_test.go
    ssh_gss_test.go
    tcpip_test.go
    testdata_test.go
    transport_test.go
)

GO_XTEST_SRCS(example_test.go)

END()

RECURSE(
    agent
    gotest
    internal
    knownhosts
    terminal
    test
    testdata
)
