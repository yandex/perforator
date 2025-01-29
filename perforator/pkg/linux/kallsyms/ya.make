GO_LIBRARY()

SRCS(
    resolver.go
    sort.go
)

GO_TEST_SRCS(resolver_test.go)

END()

RECURSE(
    gotest
)
