GO_LIBRARY()

LICENSE(BSD-2-Clause)

VERSION(v1.14.0)

SRCS(
    anchor.go
    types.go
)

GO_XTEST_SRCS(
    anchor_test.go
    types_test.go
)

END()

RECURSE(
    gotest
)
