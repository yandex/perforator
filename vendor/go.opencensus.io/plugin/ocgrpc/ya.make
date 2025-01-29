GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.24.0)

SRCS(
    client.go
    client_metrics.go
    client_stats_handler.go
    doc.go
    server.go
    server_metrics.go
    server_stats_handler.go
    stats_common.go
    trace_common.go
)

GO_TEST_SRCS(
    benchmark_test.go
    client_spec_test.go
    client_stats_handler_test.go
    grpc_test.go
    server_spec_test.go
    server_stats_handler_test.go
    trace_common_test.go
)

GO_XTEST_SRCS(
    end_to_end_test.go
    example_test.go
    trace_test.go
)

END()

RECURSE(
    # gotest
)
