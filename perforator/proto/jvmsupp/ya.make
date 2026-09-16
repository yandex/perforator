PROTO_LIBRARY()

INCLUDE(${ARCADIA_ROOT}/perforator/proto/tags.inc)

PEERDIR(perforator/agent/preprocessing/proto/jvm)

GRPC()

SRCS(
    service.proto
)

END()
