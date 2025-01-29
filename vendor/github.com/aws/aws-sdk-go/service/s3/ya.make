GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.46.7)

SRCS(
    api.go
    body_hash.go
    bucket_location.go
    customizations.go
    doc.go
    doc_custom.go
    endpoint.go
    endpoint_builder.go
    errors.go
    host_style_bucket.go
    platform_handlers_go1.6.go
    service.go
    sse.go
    statusok_error.go
    unmarshal_error.go
    waiters.go
)

GO_TEST_SRCS(
    bench_test.go
    body_hash_test.go
    endpoint_test.go
    eventstream_example_test.go
    eventstream_test.go
    unmarshal_error_leak_test.go
)

GO_XTEST_SRCS(
    bucket_location_test.go
    content_md5_test.go
    customizations_test.go
    examples_test.go
    host_style_bucket_test.go
    platform_handlers_go1.6_test.go
    sse_test.go
    statusok_error_test.go
    unmarshal_error_test.go
)

END()

RECURSE(
    gotest
    internal
    s3crypto
    s3iface
    s3manager
)
