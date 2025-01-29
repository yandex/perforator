GO_LIBRARY()

SRCS(
    cache.go
    evict.go
    writer.go
)

GO_TEST_SRCS(cache_test.go)

END()

RECURSE(
    gotest
)
