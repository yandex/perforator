GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.79.1)

SRCS(
    status.go
)

GO_XTEST_SRCS(status_test.go)

END()

RECURSE(
    gotest
)
