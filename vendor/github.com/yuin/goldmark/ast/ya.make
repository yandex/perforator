GO_LIBRARY()

LICENSE(MIT)

VERSION(v1.7.8)

SRCS(
    ast.go
    block.go
    inline.go
)

GO_TEST_SRCS(ast_test.go)

END()

RECURSE(
    gotest
)
