PROTO_LIBRARY()

INCLUDE_TAGS(GO_PROTO)

IF (OPENSOURCE)
    EXCLUDE_TAGS(GRPC)
    EXCLUDE_TAGS(JAVA_PROTO)
ENDIF()

PEERDIR(
    perforator/proto/lib/compression
)

SRCS(
    container.proto
    merge_options.proto
    profile.proto
    render_options.proto
)

END()
