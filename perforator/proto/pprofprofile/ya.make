PROTO_LIBRARY()

GRPC()

INCLUDE_TAGS(GO_PROTO)

SRCS(
    profile.proto
    lightweightprofile.proto
)

END()

IF (NOT OPENSOURCE)
    RECURSE(
        tests
    )
ENDIF()
