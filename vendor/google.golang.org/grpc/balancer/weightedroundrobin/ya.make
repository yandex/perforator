GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.73.0)

SRCS(
    balancer.go
    config.go
    logging.go
    scheduler.go
)

GO_TEST_SRCS(metrics_test.go)

GO_XTEST_SRCS(
    # balancer_test.go
)

END()

RECURSE(
    gotest
    internal
)
