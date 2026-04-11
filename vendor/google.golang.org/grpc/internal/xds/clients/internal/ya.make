GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.79.1)

SRCS(
    internal.go
)

GO_TEST_SRCS(internal_test.go)

END()

RECURSE(
    backoff
    buffer
    gotest
    pretty
    syncutil
    testutils
)
