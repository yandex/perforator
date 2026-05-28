GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.3.0)

SRCS(
    grpc.go
)

GO_TEST_SRCS(grpc_test.go)

END()

RECURSE(
    gotest
)
