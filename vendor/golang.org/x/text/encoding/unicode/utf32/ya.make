GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v0.33.0)

SRCS(
    utf32.go
)

GO_TEST_SRCS(utf32_test.go)

END()

RECURSE(
    gotest
)
