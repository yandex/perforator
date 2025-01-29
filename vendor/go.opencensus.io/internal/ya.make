GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.24.0)

SRCS(
    internal.go
    sanitize.go
    traceinternals.go
)

GO_TEST_SRCS(sanitize_test.go)

END()

RECURSE(
    gotest
    tagencoding
    testpb
)
