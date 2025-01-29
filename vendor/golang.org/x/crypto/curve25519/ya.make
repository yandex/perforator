GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v0.31.0)

SRCS(
    curve25519.go
)

GO_XTEST_SRCS(
    curve25519_test.go
    vectors_test.go
)

END()

RECURSE(
    gotest
)
