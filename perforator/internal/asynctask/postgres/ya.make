GO_LIBRARY()

SRCS(
    config.go
    row.go
    service.go
)

# This test requires library/recipes, which is not supported in the oss repo
IF (NOT OPENSOURCE)
    GO_TEST_SRCS(service_test.go)
ENDIF()

END()

IF (NOT OPENSOURCE)
    RECURSE(gotest)
ENDIF()
