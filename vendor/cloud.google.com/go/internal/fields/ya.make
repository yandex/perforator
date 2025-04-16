GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.116.0)

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
