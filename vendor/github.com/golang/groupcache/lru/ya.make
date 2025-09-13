GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.0.0-20241129210726-2c02b8208cf8)

SRCS(
    lru.go
)

GO_TEST_SRCS(lru_test.go)

END()

RECURSE(
    gotest
)
