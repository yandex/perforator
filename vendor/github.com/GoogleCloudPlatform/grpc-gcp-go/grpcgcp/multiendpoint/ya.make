GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.5.2)

SRCS(
    endpoint.go
    multiendpoint.go
)

GO_TEST_SRCS(multiendpoint_test.go)

END()

RECURSE(
    gotest
)
