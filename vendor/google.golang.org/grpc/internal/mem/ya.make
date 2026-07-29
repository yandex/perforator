GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.80.0)

SRCS(
    buffer_pool.go
)

GO_TEST_SRCS(buffer_pool_test.go)

END()

RECURSE(
    gotest
)
