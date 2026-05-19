GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.13.2)

SRCS(
    diffcopy.go
    filesync.go
    filesync.pb.go
    generate.go
)

GO_TEST_SRCS(filesync_test.go)

END()

RECURSE(
    gotest
)
