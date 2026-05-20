GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.18.2)

SRCS(
    control.pb.go
    control_grpc.pb.go
    control_vtproto.pb.go
)

GO_TEST_SRCS(control_bench_test.go)

END()

RECURSE(
    gotest
)
