GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.73.0)

SRCS(
    bdp_estimator.go
    client_stream.go
    controlbuf.go
    defaults.go
    flowcontrol.go
    handler_server.go
    http2_client.go
    http2_server.go
    http_util.go
    logging.go
    proxy.go
    server_stream.go
    transport.go
)

GO_TEST_SRCS(
    handler_server_test.go
    http_util_test.go
    keepalive_test.go
    proxy_test.go
    transport_test.go
)

GO_XTEST_SRCS(
    # proxy_ext_test.go
)

END()

RECURSE(
    gotest
    grpchttp2
    networktype
)
