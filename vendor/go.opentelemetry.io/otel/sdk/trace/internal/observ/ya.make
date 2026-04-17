GO_LIBRARY()

LICENSE(
    Apache-2.0 AND
    BSD-3-Clause
)

VERSION(v1.40.0)

SRCS(
    batch_span_processor.go
    doc.go
    simple_span_processor.go
    tracer.go
)

GO_XTEST_SRCS(
    batch_span_processor_test.go
    observ_test.go
    simple_span_processor_test.go
    tracer_test.go
)

END()

RECURSE(
    gotest
)
