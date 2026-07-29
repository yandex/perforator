GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.80.0)

SRCS(
    channelz.go
)

END()

RECURSE(
    grpc_channelz_v1
    internal
    service
)
