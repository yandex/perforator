GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.57.0)

SRCS(
    client.go
    common.go
    config.go
    doc.go
    handler.go
    labeler.go
    start_time_context.go
    transport.go
    version.go
)

GO_TEST_SRCS(
    start_time_context_test.go
    transport_example_test.go
    transport_test.go
)

GO_XTEST_SRCS(
    handler_example_test.go
    handler_test.go
)

END()

RECURSE(
    filters
    gotest
    internal
)
