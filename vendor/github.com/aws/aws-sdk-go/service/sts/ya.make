GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.46.7)

SRCS(
    api.go
    customizations.go
    doc.go
    errors.go
    service.go
)

GO_XTEST_SRCS(
    customizations_test.go
    examples_test.go
)

END()

RECURSE(
    gotest
    stsiface
)
