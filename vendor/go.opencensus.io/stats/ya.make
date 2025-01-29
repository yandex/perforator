GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.24.0)

SRCS(
    doc.go
    measure.go
    measure_float64.go
    measure_int64.go
    record.go
    units.go
)

GO_XTEST_SRCS(
    benchmark_test.go
    example_test.go
    record_test.go
)

END()

RECURSE(
    gotest
    internal
    view
)
