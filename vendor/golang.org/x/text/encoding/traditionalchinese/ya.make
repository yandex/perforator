GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v0.21.0)

SRCS(
    big5.go
    tables.go
)

GO_TEST_SRCS(all_test.go)

END()

RECURSE(
    gotest
)
