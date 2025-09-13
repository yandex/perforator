PROTO_LIBRARY()

GRPC()

INCLUDE_TAGS(GO_PROTO)

IF (OPENSOURCE)
    EXCLUDE_TAGS(JAVA_PROTO)
ENDIF()

PEERDIR(
    perforator/proto/lib/time_interval
)

SRCS(
    custom_profiling_operation.proto
)

END()
