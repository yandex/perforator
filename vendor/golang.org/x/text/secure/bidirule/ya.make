GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v0.34.0)

SRCS(
    bidirule.go
)

GO_TEST_SRCS(
    bench_test.go
    bidirule_test.go
)

END()

RECURSE(
    gotest
)
