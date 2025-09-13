GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.73.0)

SRCS(
    grpc_transport.go
)

GO_TEST_SRCS(grpc_transport_test.go)

GO_XTEST_SRCS(
    # examples_test.go
    # grpc_transport_ext_test.go
)

END()

RECURSE(
    gotest
)
