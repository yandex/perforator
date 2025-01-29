GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.46.7)

SRCS(
    copy.go
    equal.go
    path_value.go
    prettify.go
    string_value.go
)

GO_TEST_SRCS(
    prettify_test.go
    string_value_test.go
)

GO_XTEST_SRCS(
    copy_test.go
    equal_test.go
    path_value_test.go
)

END()

RECURSE(
    gotest
)
