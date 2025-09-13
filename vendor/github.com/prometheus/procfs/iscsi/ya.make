GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.16.0)

SRCS(
    get.go
    iscsi.go
)

GO_XTEST_SRCS(get_test.go)

END()

RECURSE(
    # gotest
)
