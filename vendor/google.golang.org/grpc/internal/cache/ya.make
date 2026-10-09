GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.83.2)

SRCS(
    timeoutCache.go
)

GO_TEST_SRCS(timeoutCache_test.go)

END()

RECURSE(
    gotest
)
