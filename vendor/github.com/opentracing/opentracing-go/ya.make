GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.2.0)

SRCS(
    ext.go
    globaltracer.go
    gocontext.go
    noop.go
    propagation.go
    span.go
    tracer.go
)

GO_TEST_SRCS(
    globaltracer_test.go
    gocontext_test.go
    options_test.go
    propagation_test.go
    testtracer_test.go
)

END()

RECURSE(
    ext
    gotest
    harness
    log
    mocktracer
)
