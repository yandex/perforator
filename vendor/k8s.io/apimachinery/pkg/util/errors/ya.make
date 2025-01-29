GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.26.1)

SRCS(
    doc.go
    errors.go
)

GO_TEST_SRCS(errors_test.go)

END()

RECURSE(
    gotest
)
