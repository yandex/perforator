GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.4.3)

SRCS(
    dep.go
    mixed.go
)

GO_XTEST_SRCS(dep_test.go)

END()

RECURSE(
    gotest
)
