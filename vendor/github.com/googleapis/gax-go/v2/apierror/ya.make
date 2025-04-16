GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v2.14.1)

SRCS(
    apierror.go
)

GO_TEST_SRCS(apierror_test.go)

END()

RECURSE(
    gotest
    internal
)
