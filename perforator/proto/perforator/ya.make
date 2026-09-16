PROTO_LIBRARY()

INCLUDE(${ARCADIA_ROOT}/perforator/proto/tags.inc)

GRPC()

SRCS(
    custom_profile.proto
    perforator.proto
    microscope_service.proto
    record_remote.proto
    task_service.proto
)

GO_GRPC_GATEWAY_V2_SRCS(
    perforator.proto
    task_service.proto
)

PEERDIR(
    perforator/proto/custom_profiling_operation
    perforator/proto/lib/time_interval
    perforator/proto/lib/pagination
    perforator/proto/profile
)

IF (NOT GO_PROTO)
    PEERDIR(
        contrib/libs/googleapis-common-protos
    )
ENDIF()

USE_COMMON_GOOGLE_APIS(api/annotations)

END()
