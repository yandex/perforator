GO_LIBRARY()

LICENSE(MIT)

VERSION(v1.28.0)

SRCS(
    buffer.go
    pool.go
)

GO_TEST_SRCS(
    buffer_test.go
    pool_test.go
)

END()

RECURSE(
    gotest
)
