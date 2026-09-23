GO_LIBRARY()

SRCS(
    result.go
)

GO_TEST_SRCS(result_test.go)

END()

RECURSE_FOR_TESTS(
    gotest
)
