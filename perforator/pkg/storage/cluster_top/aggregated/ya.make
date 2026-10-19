GO_LIBRARY()

SRCS(
    model.go
    snapshot.go
    snapshot_session.go
    storage.go
)

# This test requires library/recipes, which is not supported in the oss repo
IF (NOT OPENSOURCE)
    GO_TEST_SRCS(
        snapshot_session_test.go
        snapshot_test.go
        storage_test.go
    )
ENDIF()

END()

RECURSE(gotest)
