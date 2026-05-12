GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.33.0)

SRCS(
    generated.pb.go
    group_version.go
    interfaces.go
)

GO_TEST_SRCS(group_version_test.go)

END()

RECURSE(
    gotest
)
