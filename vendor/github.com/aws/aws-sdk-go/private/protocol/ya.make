GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.46.7)

SRCS(
    host.go
    host_prefix.go
    idempotency.go
    jsonvalue.go
    payload.go
    protocol.go
    timestamp.go
    unmarshal.go
    unmarshal_error.go
)

GO_TEST_SRCS(
    host_prefix_test.go
    host_test.go
    jsonvalue_test.go
    protocol_go1.7_test.go
    timestamp_test.go
)

GO_XTEST_SRCS(
    idempotency_test.go
    protocol_test.go
    unmarshal_test.go
)

END()

RECURSE(
    ec2query
    eventstream
    # gotest
    json
    jsonrpc
    query
    rest
    restjson
    restxml
    xml
)
