GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.79.1)

SRCS(
    spiffe.go
)

GO_TEST_SRCS(
    # spiffe_test.go
)

END()

RECURSE(
    gotest
)
