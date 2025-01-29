GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.24.0)

SRCS(
    basetypes.go
    config.go
    doc.go
    evictedqueue.go
    export.go
    lrumap.go
    sampling.go
    spanbucket.go
    spanstore.go
    status_codes.go
    trace.go
    trace_api.go
    trace_go11.go
)

GO_TEST_SRCS(
    benchmark_test.go
    config_test.go
    evictedqueue_test.go
    trace_test.go
)

GO_XTEST_SRCS(examples_test.go)

END()

RECURSE(
    gotest
    internal
    propagation
    tracestate
)
