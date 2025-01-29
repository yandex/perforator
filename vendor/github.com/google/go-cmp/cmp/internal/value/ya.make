GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v0.6.0)

SRCS(
    name.go
    pointer.go
    sort.go
)

GO_TEST_SRCS(name_test.go)

GO_XTEST_SRCS(
    # sort_test.go
)

END()

RECURSE(
    gotest
)
