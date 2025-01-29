GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.1.0)

SRCS(
    doc.go
    limit.go
    span.go
    tracer.go
    tracer_provider.go
)

GO_TEST_SRCS(
    example_test.go
    limit_test.go
    span_test.go
    tracer_provider_test.go
    tracer_test.go
)

END()

RECURSE(
    gotest
    internal
)
