GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v2.2.0)

SRCS(
    context.go
    grpc.go
    store.go
    ttrpc.go
)

GO_TEST_SRCS(
    context_test.go
    ttrpc_test.go
)

END()

RECURSE(
    gotest
)
