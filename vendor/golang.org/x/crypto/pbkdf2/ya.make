GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v0.31.0)

SRCS(
    pbkdf2.go
)

GO_TEST_SRCS(pbkdf2_test.go)

END()

RECURSE(
    gotest
)
