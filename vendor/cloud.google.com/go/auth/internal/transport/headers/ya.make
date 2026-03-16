GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.18.1)

SRCS(
    headers.go
)

GO_TEST_SRCS(headers_test.go)

END()

RECURSE(
    gotest
)
