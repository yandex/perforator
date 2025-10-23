GO_LIBRARY()

SRCS(
    models.go
)

GO_TEST_SRCS(
    states_test.go
)

END()

RECURSE(
    factory
    gotest
    mocks
    postgres
)
