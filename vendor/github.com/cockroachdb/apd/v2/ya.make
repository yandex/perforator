GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v2.0.2)

SRCS(
    condition.go
    const.go
    context.go
    decimal.go
    doc.go
    error.go
    form_string.go
    format.go
    loop.go
    round.go
    table.go
)

GO_TEST_SRCS(
    bench_test.go
    const_test.go
    decimal_test.go
    error_test.go
    gda_test.go
    table_test.go
)

GO_XTEST_SRCS(example_test.go)

END()

RECURSE(
    gotest
)
