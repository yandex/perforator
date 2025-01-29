GO_LIBRARY()

TAG(ya:run_go_benchmark)

DATA(
    sbr://6741361681=maps
)

SRCS(
    fs.go
    maps.go
    meminfo.go
    namespaces.go
    process.go
    scan.go
)

GO_TEST_SRCS(
    meminfo_test.go
    process_test.go
)

GO_XTEST_SRCS(parse_mappings_test.go)

GO_TEST_EMBED_PATTERN(gotest/status1.txt)

END()

RECURSE(
    gotest
)
