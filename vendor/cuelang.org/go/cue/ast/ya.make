GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.4.3)

SRCS(
    ast.go
    comments.go
    ident.go
    walk.go
)

GO_XTEST_SRCS(
    ast_test.go
    ident_test.go
)

END()

RECURSE(
    astutil
    gotest
)
