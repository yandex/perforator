GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.37.0)

SRCS(
    doc.go
    error_handler.go
    handler.go
    internal_logging.go
    metric.go
    propagation.go
    trace.go
    version.go
)

GO_TEST_SRCS(
    handler_test.go
    metric_test.go
    trace_test.go
)

GO_XTEST_SRCS(
    internal_logging_test.go
    version_test.go
)

END()

RECURSE(
    attribute
    baggage
    codes
    gotest
    internal
    propagation
    semconv
)
