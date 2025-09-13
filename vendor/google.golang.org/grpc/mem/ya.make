GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.73.0)

SRCS(
    buffer_pool.go
    buffer_slice.go
    buffers.go
)

GO_XTEST_SRCS(
    buffer_pool_test.go
    buffer_slice_test.go
    buffers_test.go
)

END()

RECURSE(
    gotest
)
