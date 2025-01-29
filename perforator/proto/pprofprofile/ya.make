PROTO_LIBRARY()

GRPC()

INCLUDE_TAGS(GO_PROTO)

SRCS(
    profile.proto
)

END()

IF (NOT OPENSOURCE)
    RECURSE(
        tests
    )
ENDIF()
