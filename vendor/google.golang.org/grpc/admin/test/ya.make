GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.73.0)

SRCS(
    utils.go
)

GO_XTEST_SRCS(admin_test.go)

END()

RECURSE(
    gotest
)
