GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.79.1)

SRCS(
    pickfirst.go
)

GO_TEST_SRCS(
    # pickfirst_test.go
)

GO_XTEST_SRCS(
    # metrics_test.go
    # pickfirst_ext_test.go
)

END()

RECURSE(
    gotest
    internal
    pickfirstleaf
)
