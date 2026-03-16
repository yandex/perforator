GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.55.0)

SRCS(
    cloudmonitoring.go
    constants.go
    error.go
    metric.go
    option.go
    version.go
)

GO_TEST_SRCS(
    cloudmonitoring_test.go
    metric_test.go
)

GO_XTEST_SRCS(example_test.go)

END()

RECURSE(
    gotest
)
