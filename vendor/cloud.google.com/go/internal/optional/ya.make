GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.116.0)

SRCS(
    optional.go
)

GO_TEST_SRCS(optional_test.go)

END()

RECURSE(
    gotest
)
