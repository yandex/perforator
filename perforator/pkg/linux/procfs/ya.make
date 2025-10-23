GO_LIBRARY()

TAG(ya:run_go_benchmark)

IF (NOT OPENSOURCE)
    DATA(
        sbr://6741361681=maps
    )
ENDIF()

SRCS(
    btime.go
    fs.go
    maps.go
    meminfo.go
    namespaces.go
    process.go
    scan.go
    stat.go
)

GO_TEST_SRCS(
    btime_test.go
    meminfo_test.go
    process_test.go
)
IF (NOT OPENSOURCE)
    GO_XTEST_SRCS(parse_mappings_test.go)
ENDIF()

GO_TEST_EMBED_PATTERN(gotest/status1.txt)

END()

RECURSE(
    gotest
)
