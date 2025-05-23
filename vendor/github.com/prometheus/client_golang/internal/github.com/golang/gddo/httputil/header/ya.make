GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.21.1)

SRCS(
    header.go
)

GO_TEST_SRCS(header_test.go)

END()

RECURSE(
    gotest
)
