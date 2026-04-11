GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v2.27.2)

SRCS(
    camel.go
)

GO_TEST_SRCS(camel_test.go)

END()

RECURSE(
    gotest
)
