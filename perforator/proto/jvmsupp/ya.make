PROTO_LIBRARY()

IF (OPENSOURCE)
    EXCLUDE_TAGS(JAVA_PROTO)
ENDIF()

PEERDIR(perforator/agent/preprocessing/proto/jvm)

GRPC()

SRCS(
    service.proto
)

END()
