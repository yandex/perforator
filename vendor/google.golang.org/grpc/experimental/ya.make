GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.63.2)

SRCS(
    experimental.go
)

GO_XTEST_SRCS(shared_buffer_pool_test.go)

END()

RECURSE(
    gotest
)
