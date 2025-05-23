GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v0.0.0-20241108190413-2d47ceb2692f)

SRCS(
    constraints.go
)

GO_TEST_SRCS(constraints_test.go)

END()

RECURSE(
    gotest
)
