GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.17.1)

SRCS(
    bits.go
    exp.go
    normal.go
    rand.go
    rng.go
)

GO_TEST_SRCS(
    arith128_test.go
    modulo_test.go
    race_test.go
    rand_test.go
)

GO_XTEST_SRCS(
    example_test.go
    regress_test.go
)

END()

RECURSE(
    gotest
)
