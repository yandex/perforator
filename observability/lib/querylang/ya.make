GO_LIBRARY()

SRCS(
    ast_repr.go
    ast.go
    helpers.go
    parser.go
)

END()

RECURSE(
    operator
    parser
)
