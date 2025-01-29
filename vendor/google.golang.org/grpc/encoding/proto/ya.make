GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.63.2)

SRCS(
    proto.go
)

GO_TEST_SRCS(
    proto_benchmark_test.go
    proto_test.go
)

END()

RECURSE(
    gotest
)
