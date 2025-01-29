GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v0.21.0)

SRCS(
    htmlindex.go
    map.go
    tables.go
)

GO_TEST_SRCS(htmlindex_test.go)

END()

RECURSE(
    gotest
)
