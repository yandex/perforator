GO_LIBRARY()

SRCS(
    builder.go
    defaultmap.go
    profile.go
    result.go
)

GO_TEST_SRCS(
    builder_test.go
    result_test.go
)

END()

RECURSE_FOR_TESTS(
    gotest
)
