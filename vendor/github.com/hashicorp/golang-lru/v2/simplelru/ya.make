GO_LIBRARY()

LICENSE(MPL-2.0)

VERSION(v2.0.7)

SRCS(
    lru.go
    lru_interface.go
)

GO_TEST_SRCS(lru_test.go)

END()

RECURSE(
    gotest
)
