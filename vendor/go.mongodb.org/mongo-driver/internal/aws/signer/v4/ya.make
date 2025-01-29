GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.17.1)

SRCS(
    header_rules.go
    request.go
    uri_path.go
    v4.go
)

GO_TEST_SRCS(v4_test.go)

END()

RECURSE(
    gotest
)
