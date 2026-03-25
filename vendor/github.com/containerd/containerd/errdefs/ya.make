GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.7.29)

SRCS(
    errors.go
    grpc.go
)

GO_TEST_SRCS(grpc_test.go)

END()

RECURSE(
    gotest
)
