GO_LIBRARY()

SRCS(
    symbolizer.go
)

GO_TEST_SRCS(symbolizer_test.go)

END()

RECURSE_FOR_TESTS(
    gotest
)
