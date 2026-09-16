PROTO_LIBRARY()

GRPC()

INCLUDE(${ARCADIA_ROOT}/perforator/proto/tags.inc)

PEERDIR(
    perforator/proto/lib/time_interval
    perforator/proto/profile
)

SRCS(
    custom_profiling_operation.proto
)

END()
