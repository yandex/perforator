GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.0.0-20250403155104-27863c87afa6)

SRCS(
    dotgraph.go
    graph.go
)

GO_TEST_SRCS(
    dotgraph_test.go
    graph_test.go
)

END()

RECURSE(
    gotest
)
