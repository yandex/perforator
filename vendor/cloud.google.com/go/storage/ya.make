GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.59.2)

SRCS(
    acl.go
    bucket.go
    client.go
    contexts.go
    copy.go
    doc.go
    dynamic_delay.go
    grpc_client.go
    grpc_dp.go
    grpc_metrics.go
    grpc_reader.go
    grpc_reader_multi_range.go
    grpc_writer.go
    hmac.go
    http_client.go
    iam.go
    invoke.go
    notifications.go
    option.go
    pcu.go
    post_policy_v4.go
    reader.go
    storage.go
    trace.go
    writer.go
)

GO_TEST_SRCS(
    acl_test.go
    bucket_test.go
    client_test.go
    conformance_test.go
    contexts_test.go
    copy_test.go
    dynamic_delay_test.go
    grpc_client_test.go
    grpc_metrics_test.go
    grpc_writer_test.go
    headers_test.go
    hmac_test.go
    http_client_test.go
    integration_test.go
    invoke_test.go
    mock_test.go
    notifications_test.go
    option_test.go
    pcu_test.go
    post_policy_v4_test.go
    reader_test.go
    retry_conformance_test.go
    storage_test.go
    trace_test.go
    writer_test.go
)

GO_XTEST_SRCS(
    example_test.go
    retry_test.go
)

END()

RECURSE(
    control
    dataflux
    experimental
    # gotest
    internal
    transfermanager
)
