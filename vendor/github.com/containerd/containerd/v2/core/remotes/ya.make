GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v2.2.0)

SRCS(
    handlers.go
    resolver.go
)

GO_TEST_SRCS(handlers_test.go)

END()

RECURSE(
    docker
    errors
    gotest
)
