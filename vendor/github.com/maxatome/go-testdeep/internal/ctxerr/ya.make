GO_LIBRARY()

LICENSE(BSD-2-Clause)

VERSION(v1.14.0)

SRCS(
    context.go
    error.go
    op_error.go
    path.go
    summary.go
)

GO_XTEST_SRCS(
    context_test.go
    error_test.go
    op_error_test.go
    path_test.go
    summary_test.go
)

END()

RECURSE(
    gotest
)
