GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v0.27.0)

SRCS(
    ucd.go
)

GO_TEST_SRCS(ucd_test.go)

GO_XTEST_SRCS(example_test.go)

END()

RECURSE(
    gotest
)
