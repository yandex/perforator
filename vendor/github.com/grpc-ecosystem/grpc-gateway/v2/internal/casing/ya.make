GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v2.29.0)

SRCS(
    camel.go
)

GO_TEST_SRCS(camel_test.go)

END()

RECURSE(
    gotest
)
