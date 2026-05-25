PROTO_LIBRARY()

GRPC()

INCLUDE_TAGS(GO_PROTO)

IF (OPENSOURCE)
    EXCLUDE_TAGS(JAVA_PROTO)
ENDIF()

PEERDIR(
    perforator/proto/pprofprofile
    perforator/proto/lib/compression
)

SRCS(
    perforator_storage.proto
)

END()
