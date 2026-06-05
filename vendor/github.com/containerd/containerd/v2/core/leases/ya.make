GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v2.2.0)

SRCS(
    context.go
    grpc.go
    id.go
    lease.go
)

GO_TEST_SRCS(lease_test.go)

END()

RECURSE(
    gotest
    proxy
)
