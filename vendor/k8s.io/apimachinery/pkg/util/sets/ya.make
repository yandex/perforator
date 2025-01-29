GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.26.1)

SRCS(
    byte.go
    doc.go
    empty.go
    int.go
    int32.go
    int64.go
    ordered.go
    set.go
    string.go
)

GO_TEST_SRCS(set_test.go)

GO_XTEST_SRCS(set_generic_test.go)

END()

RECURSE(
    gotest
)
