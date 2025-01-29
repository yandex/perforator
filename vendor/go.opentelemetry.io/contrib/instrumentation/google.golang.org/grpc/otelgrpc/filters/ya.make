GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.54.0)

SRCS(
    filters.go
)

GO_TEST_SRCS(filters_test.go)

END()

RECURSE(
    gotest
    interceptor
)
