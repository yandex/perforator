GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.26.1)

SRCS(
    doc.go
    fields.go
    requirements.go
    selector.go
)

GO_TEST_SRCS(
    fields_test.go
    selector_test.go
)

END()

RECURSE(
    gotest
)
