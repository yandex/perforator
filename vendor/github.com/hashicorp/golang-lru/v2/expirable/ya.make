GO_LIBRARY()

LICENSE(MPL-2.0)

VERSION(v2.0.7)

SRCS(
    expirable_lru.go
)

GO_TEST_SRCS(expirable_lru_test.go)

END()

RECURSE(
    gotest
)
