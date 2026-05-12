GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.33.0)

SRCS(
    validation.go
)

GO_TEST_SRCS(validation_test.go)

END()

RECURSE(
    gotest
)
