GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v0.38.0)

SRCS(
    catalog.go
    dict.go
)

GO_TEST_SRCS(catalog_test.go)

END()

RECURSE(
    gotest
)
