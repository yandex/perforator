GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v0.0.0-20260410095643-746e56fc9e2f)

SRCS(
    slices.go
    sort.go
)

GO_TEST_SRCS(
    slices_test.go
    sort_benchmark_test.go
    sort_test.go
)

END()

RECURSE(
    gotest
)
