GO_LIBRARY()

SRCS(
    config.go
    models.go
    query.go
    storage.go
)

GO_TEST_SRCS(query_test.go)

END()

RECURSE(
    gotest
)
