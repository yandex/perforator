GO_LIBRARY()

SRCS(
    row.go
    storage.go
)

# This test requires library/recipes, which is not supported in the oss repo
IF (NOT OPENSOURCE)
    GO_TEST_SRCS(storage_test.go)
ENDIF()

END()

IF (NOT OPENSOURCE)
    RECURSE(gotest)
ENDIF()
