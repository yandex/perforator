GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.0.0-20240116145035-ef3ab179eed6)

SRCS(
    ast.go
)

GO_XTEST_SRCS(ast_test.go)

END()

RECURSE(
    gotest
)
