GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v0.176.1)

SRCS(
    http.go
)

GO_TEST_SRCS(http_test.go)

END()

RECURSE(
    gotest
)
