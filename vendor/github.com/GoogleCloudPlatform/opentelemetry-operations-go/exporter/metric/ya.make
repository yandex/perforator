GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.51.0)

SRCS(
    cloudmonitoring.go
    constants.go
    error.go
    metric.go
    option.go
    version.go
)

GO_TEST_SRCS(metric_test.go)

GO_XTEST_SRCS(example_test.go)

END()

RECURSE(
    gotest
)
