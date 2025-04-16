GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v0.0.0-20240903120638-7835f813f4da)

SRCS(
    adaptor.go
    doc.go
    errors.go
    fmt.go
    format.go
    frame.go
    wrap.go
)

GO_TEST_SRCS(fmt_unexported_test.go)

GO_XTEST_SRCS(
    errors_test.go
    example_As_test.go
    example_FormatError_test.go
    example_test.go
    fmt_test.go
    stack_test.go
    wrap_test.go
)

END()

RECURSE(
    gotest
    internal
)
