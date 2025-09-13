GO_LIBRARY()

LICENSE(BSD-2-Clause)

VERSION(v1.14.0)

SRCS(
    lex.go
    marshal.go
    parser.go
)

GO_XTEST_SRCS(
    marshal_test.go
    parser_test.go
)

END()

RECURSE(
    gotest
)
