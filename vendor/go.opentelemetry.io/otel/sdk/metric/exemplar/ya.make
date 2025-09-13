GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.36.0)

SRCS(
    doc.go
    exemplar.go
    filter.go
    fixed_size_reservoir.go
    histogram_reservoir.go
    reservoir.go
    storage.go
    value.go
)

GO_TEST_SRCS(
    filter_test.go
    fixed_size_reservoir_test.go
    histogram_reservoir_test.go
    reservoir_test.go
    value_test.go
)

END()

RECURSE(
    gotest
)
