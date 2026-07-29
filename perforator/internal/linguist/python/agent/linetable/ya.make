GO_LIBRARY()

SRCS(
    cache.go
    linetable.go
)

GO_TEST_SRCS(
    cache_test.go
    linetable_test.go
)

END()

RECURSE_FOR_TESTS(
    gotest
)
