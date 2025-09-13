GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v0.27.0)

SRCS(
    bidirule.go
    bidirule10.0.0.go
)

GO_TEST_SRCS(
    bench_test.go
    bidirule10.0.0_test.go
    bidirule_test.go
)

END()

RECURSE(
    gotest
)
