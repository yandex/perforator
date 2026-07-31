GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.80.0)

# GO_TEST_SRCS(buffer_pool_test.go)

SRCS(
    buffer_pool.go
)

GO_TEST_SRCS(buffer_pool_test.go)

END()

RECURSE(
    gotest
)
