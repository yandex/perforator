GO_LIBRARY()

LICENSE(MIT)

VERSION(v1.101.0)

SRCS(
    checker.go
    comment.go
    error.go
    format.go
    iter.go
    patch.go
    quicktest.go
    report.go
)

GO_TEST_SRCS(export_test.go)

GO_XTEST_SRCS(
    checker_test.go
    comment_test.go
    error_test.go
    example_test.go
    format_test.go
    quicktest_test.go
    report_test.go
)

END()

RECURSE(
    # gotest
)
