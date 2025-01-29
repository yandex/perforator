GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v0.176.1)

SRCS(
    iterator.go
)

GO_XTEST_SRCS(
    examples_test.go
    iterator_test.go
)

END()

RECURSE(
    gotest
    testing
)
