GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.4.3)

SRCS(
    list.go
    math.go
    pkg.go
    sort.go
)

GO_XTEST_SRCS(list_test.go)

END()

RECURSE(
    gotest
)
