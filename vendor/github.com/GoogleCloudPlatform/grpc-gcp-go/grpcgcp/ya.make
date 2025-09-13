GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.5.2)

SRCS(
    doc.go
    gcp_balancer.go
    gcp_interceptor.go
    gcp_logger.go
    gcp_multiendpoint.go
    gcp_picker.go
)

GO_TEST_SRCS(
    # gcp_balancer_test.go
    gcp_interceptor_test.go
    # gcp_picker_test.go
)

END()

RECURSE(
    gotest
    grpc_gcp
    mocks
    multiendpoint
)
