GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.26.1)

SRCS(
    validation.go
)

GO_TEST_SRCS(validation_test.go)

END()

RECURSE(
    gotest
)
