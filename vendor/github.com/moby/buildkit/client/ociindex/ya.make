GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.23.2)

SRCS(
    ociindex.go
)

GO_TEST_SRCS(ociindex_test.go)

END()

RECURSE(
    gotest
)
