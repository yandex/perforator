GO_LIBRARY()

SRCS(
    models.go
    lease.go
    test_suite.go
)

GO_TEST_SRCS(
    lease_test.go
)

END()

RECURSE(
    gotest
    postgres
)
