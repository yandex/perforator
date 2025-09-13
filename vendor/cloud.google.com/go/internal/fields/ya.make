GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.120.0)

GO_SKIP_TESTS(TestFieldsWithTags)

SRCS(
    fields.go
    fold.go
)

GO_TEST_SRCS(
    fields_test.go
    fold_test.go
)

END()

RECURSE(
    gotest
)
