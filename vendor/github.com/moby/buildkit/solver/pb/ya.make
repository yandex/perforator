GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.13.2)

SRCS(
    attr.go
    caps.go
    const.go
    generate.go
    json.go
    ops.pb.go
    platform.go
)

GO_TEST_SRCS(json_test.go)

END()

RECURSE(
    gotest
)
