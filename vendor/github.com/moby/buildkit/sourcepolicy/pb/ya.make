GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.13.2)

SRCS(
    generate.go
    json.go
    policy.pb.go
)

GO_TEST_SRCS(json_test.go)

END()

RECURSE(
    gotest
)
