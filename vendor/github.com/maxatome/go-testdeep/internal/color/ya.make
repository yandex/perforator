GO_LIBRARY()

LICENSE(BSD-2-Clause)

VERSION(v1.14.0)

SRCS(
    color.go
)

GO_XTEST_SRCS(color_test.go)

END()

RECURSE(
    gotest
)
