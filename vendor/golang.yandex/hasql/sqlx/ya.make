GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.1.1)

SRCS(
    cluster.go
    forward.go
    node.go
)

GO_TEST_SRCS(
    cluster_test.go
    node_test.go
)

END()

RECURSE(
    gotest
)
