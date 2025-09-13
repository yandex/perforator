GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.73.0)

SRCS(
    pickfirstleaf.go
)

GO_TEST_SRCS(pickfirstleaf_test.go)

GO_XTEST_SRCS(
    # metrics_test.go
    # pickfirstleaf_ext_test.go
)

END()

RECURSE(
    gotest
)
