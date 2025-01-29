GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v0.31.0)

SRCS(
    rc2.go
)

GO_TEST_SRCS(
    bench_test.go
    rc2_test.go
)

END()

RECURSE(
    gotest
)
