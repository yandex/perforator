GO_LIBRARY()

LICENSE(BSD-2-Clause)

VERSION(v1.14.0)

SRCS(
    visited.go
)

GO_XTEST_SRCS(visited_test.go)

END()

RECURSE(
    gotest
)
