GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.63.2)

SRCS(
    bundle.go
)

GO_TEST_SRCS(
    # bundle_test.go
)

GO_XTEST_SRCS(bundle_ext_test.go)

END()

RECURSE(
    gotest
)
