GO_LIBRARY()
SRCS(
    cluster_top.go
)

GO_TEST_SRCS(cluster_top_test.go)

END()

RECURSE(
    gotest
    mocks
)
