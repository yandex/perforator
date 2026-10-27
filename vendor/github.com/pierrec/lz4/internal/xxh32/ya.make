GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v2.6.1+incompatible)

SRCS(
    xxh32zero.go
)

GO_XTEST_SRCS(xxh32zero_test.go)

END()

RECURSE(
    gotest
)
