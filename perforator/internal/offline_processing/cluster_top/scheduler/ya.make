GO_LIBRARY()

PEERDIR(
    perforator/pkg/profilequerylang
    perforator/pkg/sampletype
)

SRCS(
    discover_jobs.go
    filters.go
    scheduler.go
)

IF (NOT OPENSOURCE)
    GO_TEST_SRCS(
        discover_jobs_test.go
    )
ENDIF()

END()

IF (NOT OPENSOURCE)
    RECURSE(gotest)
ENDIF()
