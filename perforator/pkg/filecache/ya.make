GO_LIBRARY()

SRCS(
    filecache.go
)

GO_TEST_SRCS(
    filecache_test.go
)

END()

RECURSE(
    gotest
)
