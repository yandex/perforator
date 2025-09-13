GO_LIBRARY()

LICENSE(BSD-2-Clause)

VERSION(v1.14.0)

SRCS(
    slice.go
)

GO_XTEST_SRCS(slice_test.go)

END()

RECURSE(
    gotest
)
