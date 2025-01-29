GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.54.0)

SRCS(
    parse.go
)

GO_TEST_SRCS(parse_test.go)

END()

RECURSE(
    gotest
    test
)
