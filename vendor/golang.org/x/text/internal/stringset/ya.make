GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v0.35.0)

SRCS(
    set.go
)

GO_TEST_SRCS(set_test.go)

END()

RECURSE(
    gotest
)
