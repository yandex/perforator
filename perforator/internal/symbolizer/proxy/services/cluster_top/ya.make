GO_LIBRARY()
SRCS(
    config.go
    query.go
    cluster_top.go
)

GO_TEST_SRCS(
    cluster_top_test.go
    query_test.go
)

END()

RECURSE(
    gotest
    mocks
)
