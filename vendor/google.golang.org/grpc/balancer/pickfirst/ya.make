GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.73.0)

SRCS(
    pickfirst.go
)

GO_TEST_SRCS(pickfirst_test.go)

GO_XTEST_SRCS(
    # pickfirst_ext_test.go
)

END()

RECURSE(
    gotest
    internal
    pickfirstleaf
)
