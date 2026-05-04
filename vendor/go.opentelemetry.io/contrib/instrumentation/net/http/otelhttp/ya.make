GO_LIBRARY()

LICENSE(
    Apache-2.0 AND
    BSD-3-Clause
)

VERSION(v0.63.0)

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
    handler_test.go
    start_time_context_test.go
    transport_example_test.go
    transport_test.go
)

GO_XTEST_SRCS(
    client_test.go
    config_test.go
    handler_example_test.go
    version_test.go
)

END()

RECURSE(
    filters
    gotest
    internal
)
