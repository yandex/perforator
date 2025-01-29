GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.24.0)

SRCS(
    client.go
    client_stats.go
    doc.go
    route.go
    server.go
    span_annotating_client_trace.go
    stats.go
    trace.go
    wrapped_body.go
)

GO_TEST_SRCS(
    propagation_test.go
    server_test.go
    stats_test.go
    # trace_test.go
)

GO_XTEST_SRCS(
    client_test.go
    example_test.go
    route_test.go
    span_annotating_client_trace_test.go
)

END()

RECURSE(
    gotest
    propagation
)
