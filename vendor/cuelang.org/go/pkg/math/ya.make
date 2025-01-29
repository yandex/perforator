GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.4.3)

SRCS(
    big.go
    manual.go
    math.go
    pkg.go
)

GO_XTEST_SRCS(math_test.go)

END()

RECURSE(
    bits
    gotest
)
