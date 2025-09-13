GO_LIBRARY()

SRCS(
    models.go
    storage.go
)

IF (NOT OPENSOURCE)
    GO_TEST_SRCS(storage_test.go)
ENDIF()

END()

RECURSE(
    gotest
)
