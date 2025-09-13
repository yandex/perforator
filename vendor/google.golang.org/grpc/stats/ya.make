GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.73.0)

SRCS(
    handlers.go
    metrics.go
    stats.go
)

GO_XTEST_SRCS(stats_test.go)

END()

RECURSE(
    gotest
    opentelemetry
    # yo
)
