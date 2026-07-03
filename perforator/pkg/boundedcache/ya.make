GO_LIBRARY()

SRCS(
    boundedcache.go
)

GO_TEST_SRCS(
    boundedcache_test.go
)

END()

RECURSE(
    gotest
)
