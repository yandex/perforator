GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.5.0)

SRCS(
    nat.go
    parse.go
    sort.go
)

GO_TEST_SRCS(
    nat_test.go
    parse_test.go
    sort_test.go
)

END()

RECURSE(
    gotest
)
