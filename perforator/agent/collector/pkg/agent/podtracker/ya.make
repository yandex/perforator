GO_LIBRARY()

SRCS(
    filter.go
)

GO_TEST_SRCS(
    filter_test.go
)

END()

RECURSE_FOR_TESTS(gotest)
