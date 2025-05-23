GO_LIBRARY()

LICENSE(MIT)

VERSION(v1.4.0)

SRCS(
    const.go
    decimal-go.go
    decimal.go
    rounding.go
)

GO_TEST_SRCS(
    const_test.go
    decimal_bench_test.go
    decimal_test.go
)

END()

RECURSE(
    gotest
)
