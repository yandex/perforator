GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.4.3)

SRCS(
    errors.go
)

GO_TEST_SRCS(errors_test.go)

END()

RECURSE(
    gotest
)
