GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.1.1)

SRCS(
    check_nodes.go
    cluster.go
    cluster_opts.go
    errors_collector.go
    node.go
    node_pickers.go
    trace.go
)

GO_TEST_SRCS(
    check_nodes_test.go
    cluster_opts_test.go
    cluster_test.go
    errors_collector_test.go
    node_pickers_test.go
)

GO_XTEST_SRCS(
    example_cluster_test.go
    example_trace_test.go
)

END()

RECURSE(
    checkers
    gotest
    sqlx
)
