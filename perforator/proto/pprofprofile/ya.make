PROTO_LIBRARY()

GRPC()

INCLUDE(${ARCADIA_ROOT}/perforator/proto/tags.inc)

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
