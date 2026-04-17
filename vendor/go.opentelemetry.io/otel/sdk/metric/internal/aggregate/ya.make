GO_LIBRARY()

LICENSE(
    Apache-2.0 AND
    BSD-3-Clause
)

VERSION(v1.40.0)

SRCS(
    aggregate.go
    atomic.go
    doc.go
    drop.go
    exemplar.go
    exponential_histogram.go
    filtered_reservoir.go
    histogram.go
    lastvalue.go
    limit.go
    sum.go
)

GO_TEST_SRCS(
    aggregate_test.go
    atomic_test.go
    drop_test.go
    exemplar_test.go
    exponential_histogram_test.go
    filtered_reservoir_test.go
    histogram_test.go
    lastvalue_test.go
    limit_test.go
    sum_test.go
)

END()

RECURSE(
    gotest
)
