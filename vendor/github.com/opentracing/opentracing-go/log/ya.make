GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.2.0)

SRCS(
    field.go
    util.go
)

GO_TEST_SRCS(
    field_test.go
    util_test.go
)

END()

RECURSE(
    gotest
)
