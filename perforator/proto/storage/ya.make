PROTO_LIBRARY()

GRPC()

INCLUDE(${ARCADIA_ROOT}/perforator/proto/tags.inc)

PEERDIR(
    perforator/proto/pprofprofile
    perforator/proto/lib/compression
)

SRCS(
    perforator_storage.proto
)

END()
