GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.0.0)

SRCS(
    normalize_url.go
)

GO_TEST_SRCS(normalize_url_test.go)

END()

RECURSE(
    gotest
)
