GO_LIBRARY()

SRCS(
    config.go
    query.go
    query_builder.go
    query_config.go
    model.go
    snapshot.go
    snapshot_session.go
    storage.go
)

GO_TEST_SRCS(
    query_builder_test.go
)

# This test requires library/recipes, which is not supported in the oss repo
IF (NOT OPENSOURCE)
    GO_TEST_SRCS(
        query_test.go
        serving_table_test.go
        snapshot_session_test.go
        snapshot_test.go
        storage_test.go
    )
ENDIF()

END()

RECURSE(gotest)
