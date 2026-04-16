GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.79.3)

SRCS(
    experimental.go
)

GO_XTEST_SRCS(shared_buffer_pool_test.go)

END()

RECURSE(
    credentials
    gotest
    opentelemetry
    stats
)
