GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.2.0)

SRCS(
    field.go
    tags.go
)

GO_XTEST_SRCS(
    field_test.go
    tags_test.go
)

END()

RECURSE(
    gotest
)
