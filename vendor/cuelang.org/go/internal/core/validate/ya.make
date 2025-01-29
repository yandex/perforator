GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.4.3)

SRCS(
    validate.go
)

GO_TEST_SRCS(validate_test.go)

END()

RECURSE(
    gotest
)
