GO_LIBRARY()

LICENSE(BSD-2-Clause)

VERSION(v1.14.0)

SRCS(
    hooks.go
)

GO_XTEST_SRCS(hooks_test.go)

END()

RECURSE(
    gotest
)
