GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.23.2)

SRCS(
    attr.go
    caps.go
    const.go
    json.go
    ops.go
    ops.pb.go
    ops_vtproto.pb.go
    platform.go
)

GO_TEST_SRCS(json_test.go)

END()

RECURSE(
    gotest
)
