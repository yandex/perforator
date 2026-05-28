GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.20.2)

SRCS(
    json.go
    policy.pb.go
    policy_vtproto.pb.go
)

GO_TEST_SRCS(json_test.go)

END()

RECURSE(
    gotest
)
