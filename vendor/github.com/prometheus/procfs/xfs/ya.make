GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.19.2)

SRCS(
    parse.go
    xfs.go
)

GO_XTEST_SRCS(
    parse_test.go
    xfs_test.go
)

END()

RECURSE(
    # gotest
)
