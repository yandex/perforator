GO_LIBRARY()

SRCS(
    committer.go
    pg.go
    row.go
)

# This test requires library/recipes, which is not supported in the oss repo
IF (NOT OPENSOURCE)
    GO_TEST_SRCS(pg_test.go)
ENDIF()

END()

IF (NOT OPENSOURCE)
    RECURSE(gotest)
ENDIF()
