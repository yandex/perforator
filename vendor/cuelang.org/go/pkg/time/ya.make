GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.4.3)

TAG(ya:go_no_subtest_report)

SRCS(
    duration.go
    pkg.go
    time.go
)

GO_TEST_SRCS(
    duration_test.go
    time_test.go
)

GO_XTEST_SRCS(
    # builtin_test.go
)

END()

RECURSE(
    gotest
)
