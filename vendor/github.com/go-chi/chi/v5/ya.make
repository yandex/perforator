GO_LIBRARY()

LICENSE(MIT)

VERSION(v5.2.2)

SRCS(
    chain.go
    chi.go
    context.go
    mux.go
    path_value.go
    tree.go
)

GO_TEST_SRCS(
    context_test.go
    mux_test.go
    path_value_test.go
    tree_test.go
)

END()

RECURSE(
    gotest
    middleware
)
