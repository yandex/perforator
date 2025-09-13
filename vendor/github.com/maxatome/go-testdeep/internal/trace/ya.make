GO_LIBRARY()

LICENSE(BSD-2-Clause)

VERSION(v1.14.0)

SRCS(
    stack.go
    trace.go
)

GO_XTEST_SRCS(
    stack_test.go
    trace_test.go
)

END()

RECURSE(
    gotest
)
