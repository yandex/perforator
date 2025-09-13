GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.73.0)

SRCS(
    adapt.go
    serverreflection.go
)

END()

RECURSE(
    grpc_reflection_v1
    grpc_reflection_v1alpha
    grpc_testing
    internal
    test
    # yo
)
