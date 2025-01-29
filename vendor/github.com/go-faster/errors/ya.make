GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v0.7.1)

SRCS(
    adaptor.go
    doc.go
    errors.go
    format.go
    frame.go
    into.go
    join_go120.go
    must.go
    trace.go
    wrap.go
)

GO_TEST_SRCS(
    bench_test.go
    cause_test.go
    must_test.go
)

GO_XTEST_SRCS(
    errors_test.go
    example_As_test.go
    example_FormatError_test.go
    example_Into_test.go
    example_Must_test.go
    example_test.go
    format_test.go
    join_go120_test.go
    multi_test.go
    wrap_test.go
)

END()

RECURSE(
    gotest
)
