GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v0.9.1)

SRCS(
    dec.go
    rounder.go
)

GO_TEST_SRCS(
    benchmark_test.go
    dec_go1_2_test.go
    dec_internal_test.go
)

GO_XTEST_SRCS(
    dec_test.go
    example_test.go
    rounder_example_test.go
    rounder_test.go
)

END()

RECURSE(
    gotest
)
