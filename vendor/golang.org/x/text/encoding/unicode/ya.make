GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v0.21.0)

SRCS(
    override.go
    unicode.go
)

GO_TEST_SRCS(unicode_test.go)

END()

RECURSE(
    gotest
    utf32
)
