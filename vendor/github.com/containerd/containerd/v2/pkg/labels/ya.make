GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v2.1.5)

SRCS(
    labels.go
    validate.go
)

GO_TEST_SRCS(validate_test.go)

END()

RECURSE(
    gotest
)
