GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.33.0)

SRCS(
    config.go
    context.go
    doc.go
    nonrecording.go
    noop.go
    provider.go
    span.go
    trace.go
    tracer.go
    tracestate.go
)

GO_TEST_SRCS(
    config_test.go
    context_test.go
    noop_test.go
    span_test.go
    trace_test.go
    tracestate_benchkmark_test.go
    tracestate_test.go
)

END()

RECURSE(
    embedded
    gotest
    noop
)
