GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.21.1)

SRCS(
    fs.go
)

GO_TEST_SRCS(
    # fs_test.go
)

END()

RECURSE(
    gotest
)
