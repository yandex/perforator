PROTO_LIBRARY()

GRPC()

INCLUDE_TAGS(GO_PROTO)

PEERDIR(
    perforator/proto/pprofprofile
)

SRCS(
    perforator_storage.proto
)

END()
