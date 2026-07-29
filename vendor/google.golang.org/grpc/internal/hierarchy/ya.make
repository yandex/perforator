GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.80.0)

SRCS(
    hierarchy.go
)

GO_XTEST_SRCS(hierarchy_ext_test.go)

END()

RECURSE(
    gotest
)
