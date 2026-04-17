GO_LIBRARY()

LICENSE(
    Apache-2.0 AND
    BSD-3-Clause
)

VERSION(v1.40.0)

SRCS(
    batch_span_processor.go
    doc.go
    event.go
    evictedqueue.go
    id_generator.go
    link.go
    provider.go
    sampler_env.go
    sampling.go
    simple_span_processor.go
    snapshot.go
    span.go
    span_exporter.go
    span_limits.go
    span_processor.go
    tracer.go
)

GO_TEST_SRCS(
    batch_span_processor_test.go
    evictedqueue_test.go
    id_generator_test.go
    main_test.go
    provider_test.go
    sampling_test.go
    simple_span_processor_test.go
    span_limits_test.go
    span_processor_annotator_example_test.go
    span_processor_filter_example_test.go
    span_processor_test.go
    span_test.go
    trace_test.go
    util_test.go
)

GO_XTEST_SRCS(
    # benchmark_test.go
)

END()

RECURSE(
    gotest
    internal
    tracetest
)
