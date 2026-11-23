GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.37.2)

SRCS(
    doc.go
    validation.go
    zz_generated.validations.go
)

GO_TEST_SRCS(validation_test.go)

END()

RECURSE(
    gotest
)
