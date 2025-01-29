GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v0.12.2)

SRCS(
    decimal.go
)

GO_TEST_SRCS(decimal_test.go)

END()

RECURSE(
    gotest
)
