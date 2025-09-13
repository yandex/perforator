GO_LIBRARY()

SRCS(
    config.go
    models.go
    query.go
    storage.go
)

GO_TEST_SRCS(query_test.go)

# This test requires library/recipes, which is not supported in the oss repo
IF (NOT OPENSOURCE)
    GO_TEST_SRCS(storage_test.go)
ENDIF()

END()

RECURSE(
    gotest
)
