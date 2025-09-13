GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.7.0)

SRCS(
    lru.go
)

GO_TEST_SRCS(lru_test.go)

END()

RECURSE(
    gotest
)
