GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.2.0)

SRCS(
    api_checkers.go
)

GO_TEST_SRCS(noop_api_test.go)

END()

RECURSE(
    gotest
)
