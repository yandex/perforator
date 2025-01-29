GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.40.0)

SRCS(
    acl.go
    bucket.go
    client.go
    copy.go
    doc.go
    grpc_client.go
    hmac.go
    http_client.go
    iam.go
    invoke.go
    notifications.go
    option.go
    post_policy_v4.go
    reader.go
    storage.go
    writer.go
)

GO_TEST_SRCS(
    acl_test.go
    bucket_test.go
    client_test.go
    conformance_test.go
    copy_test.go
    grpc_client_test.go
    headers_test.go
    hmac_test.go
    integration_test.go
    invoke_test.go
    mock_test.go
    notifications_test.go
    option_test.go
    post_policy_v4_test.go
    reader_test.go
    retry_conformance_test.go
    storage_test.go
    writer_test.go
)

GO_XTEST_SRCS(
    example_test.go
    retry_test.go
)

END()

RECURSE(
    control
    # gotest
    internal
)
