GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.24.0)

SRCS(
    aggregation.go
    aggregation_data.go
    collector.go
    doc.go
    export.go
    view.go
    view_to_metric.go
    worker.go
    worker_commands.go
)

GO_TEST_SRCS(
    aggregation_data_test.go
    benchmark_test.go
    collector_test.go
    view_measure_test.go
    view_test.go
    view_to_metric_test.go
    worker_test.go
)

GO_XTEST_SRCS(example_test.go)

END()

RECURSE(
    gotest
)
