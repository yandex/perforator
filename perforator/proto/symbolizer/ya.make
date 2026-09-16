PROTO_LIBRARY()

INCLUDE(${ARCADIA_ROOT}/perforator/proto/tags.inc)

GRPC()

SRCS(
    symbolizer.proto
)


END()
