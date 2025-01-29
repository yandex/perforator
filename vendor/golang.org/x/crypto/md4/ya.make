GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v0.31.0)

SRCS(
    md4.go
    md4block.go
)

GO_TEST_SRCS(md4_test.go)

GO_XTEST_SRCS(example_test.go)

END()

RECURSE(
    gotest
)
