GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v1.6.0)

SRCS(
    lru.go
)

GO_TEST_SRCS(lru_test.go)

END()

RECURSE(
    gotest
)
