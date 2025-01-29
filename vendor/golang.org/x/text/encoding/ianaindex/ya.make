GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v0.21.0)

SRCS(
    ascii.go
    ianaindex.go
    tables.go
)

GO_TEST_SRCS(
    ascii_test.go
    ianaindex_test.go
)

GO_XTEST_SRCS(example_test.go)

END()

RECURSE(
    gotest
)
