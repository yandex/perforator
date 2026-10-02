GO_LIBRARY()

SRCS(
    storage.go
    row_storage.go
)

GO_TEST_SRCS(storage_test.go)

END()

IF (NOT OPENSOURCE)
    RECURSE_FOR_TESTS(gotest)
ENDIF()
