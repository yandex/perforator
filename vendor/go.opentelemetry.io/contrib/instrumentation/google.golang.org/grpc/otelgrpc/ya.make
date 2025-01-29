GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.54.0)

SRCS(
    config.go
    doc.go
    interceptor.go
    interceptorinfo.go
    metadata_supplier.go
    semconv.go
    stats_handler.go
    version.go
)

GO_TEST_SRCS(
    config_test.go
    metadata_supplier_test.go
)

GO_XTEST_SRCS(
    benchmark_test.go
    example_test.go
)

END()

RECURSE(
    filters
    gotest
    internal
)
