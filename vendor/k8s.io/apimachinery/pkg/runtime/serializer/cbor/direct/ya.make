GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.33.0)

SRCS(
    direct.go
)

GO_XTEST_SRCS(direct_test.go)

END()

RECURSE(
    gotest
)
