GO_LIBRARY()

SRCS(
    builder.go
    defaultmap.go
    profile.go
)

GO_TEST_SRCS(builder_test.go)

END()

RECURSE_FOR_TESTS(
    gotest
)
