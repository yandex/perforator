GO_LIBRARY()

GO_XTEST_EMBED_PATTERN(testdata/traces.json)

SRCS(
    registry.go
    read.go
    table.go
)

GO_XTEST_SRCS(
    registry_test.go
    read_test.go
    table_test.go
)

END()

RECURSE_FOR_TESTS(gotest)
