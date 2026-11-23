GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.21.1)

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
