GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.79.3)

SRCS(
    testdata.go
)

END()

RECURSE(
    grpc_testing_not_regenerated
)
