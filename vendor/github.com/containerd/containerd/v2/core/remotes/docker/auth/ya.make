GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v2.2.0)

SRCS(
    fetch.go
    parse.go
)

GO_TEST_SRCS(
    fetch_test.go
    parse_test.go
)

END()

RECURSE(
    gotest
)
