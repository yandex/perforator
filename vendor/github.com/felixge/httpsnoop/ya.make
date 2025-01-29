GO_LIBRARY()

LICENSE(MIT)

VERSION(v1.0.4)

SRCS(
    capture_metrics.go
    docs.go
    wrap_generated_gteq_1.8.go
)

GO_TEST_SRCS(
    bench_test.go
    capture_metrics_test.go
    unwrap_test.go
    wrap_generated_gteq_1.8_test.go
    wrap_test.go
)

END()

RECURSE(
    gotest
)
