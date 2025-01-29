GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.24.0)

SRCS(
    resource.go
)

GO_TEST_SRCS(resource_test.go)

END()

RECURSE(
    gotest
)
