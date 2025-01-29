GO_TEST()

LICENSE(Apache-2.0)

VERSION(v2.120.1)

GO_SKIP_TESTS(TestDestinationsWithDifferentFlags)

GO_XTEST_SRCS(klog_test.go)

END()

RECURSE(
    internal
)
