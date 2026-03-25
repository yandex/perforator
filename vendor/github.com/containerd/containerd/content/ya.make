GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.7.29)

SRCS(
    adaptor.go
    content.go
    helpers.go
)

GO_TEST_SRCS(
    adaptor_test.go
    helpers_test.go
)

END()

RECURSE(
    gotest
    local
    proxy
    testsuite
)
