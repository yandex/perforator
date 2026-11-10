GO_LIBRARY()

SRCS(
    attributes.go
    upload_claim.go
    pg.go
    row.go
)

GO_TEST_SRCS(row_test.go)

# This test requires library/recipes, which is not supported in the oss repo
IF (NOT OPENSOURCE)
    GO_TEST_SRCS(pg_test.go)
ENDIF()

END()

IF (NOT OPENSOURCE)
    RECURSE(gotest)
ENDIF()
