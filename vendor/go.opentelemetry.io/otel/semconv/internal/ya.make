GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.37.0)

SRCS(
    http.go
)

GO_TEST_SRCS(http_test.go)

END()

RECURSE(
    gotest
    v2
    v3
    v4
)
