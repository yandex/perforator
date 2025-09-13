GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v0.27.0)

SRCS(
    doc.go
)

GO_TEST_SRCS(unicode_test.go)

END()

RECURSE(
    gotest
)
