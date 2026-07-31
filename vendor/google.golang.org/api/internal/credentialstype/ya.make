GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v0.272.0)

SRCS(
    credentialstype.go
)

GO_TEST_SRCS(credentialstype_test.go)

END()

RECURSE(
    gotest
)
