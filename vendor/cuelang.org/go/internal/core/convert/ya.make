GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.4.3)

SRCS(
    go.go
)

GO_XTEST_SRCS(go_test.go)

END()

RECURSE(
    gotest
)
