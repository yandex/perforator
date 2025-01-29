GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v0.12.2)

SRCS(
    parser.go
)

GO_TEST_SRCS(parser_test.go)

END()

RECURSE(
    gotest
)
