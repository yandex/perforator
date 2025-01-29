GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.46.7)

SRCS(
    doc.go
    enable.go
    metric.go
    metric_chan.go
    metric_exception.go
    reporter.go
)

GO_TEST_SRCS(
    address_test.go
    enable_test.go
    metric_chan_test.go
    metric_test.go
    reporter_internal_test.go
)

GO_XTEST_SRCS(
    example_test.go
    reporter_test.go
)

END()

RECURSE(
    # gotest
)
