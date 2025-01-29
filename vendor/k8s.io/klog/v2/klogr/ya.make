GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v2.120.1)

SRCS(
    klogr.go
)

GO_TEST_SRCS(klogr_test.go)

GO_XTEST_SRCS(output_test.go)

END()

RECURSE(
    calldepth-test
    gotest
)
