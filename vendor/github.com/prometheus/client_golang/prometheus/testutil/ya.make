GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.21.1)

SRCS(
    lint.go
    testutil.go
)

GO_TEST_SRCS(
    lint_test.go
    testutil_test.go
)

END()

RECURSE(
    gotest
    promlint
)
