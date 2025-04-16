GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.12.0)

SRCS(
    file.go
    fs.go
)

GO_TEST_SRCS(tarfs_test.go)

END()

RECURSE(
    gotest
)
