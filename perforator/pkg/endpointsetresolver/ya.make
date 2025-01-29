GO_LIBRARY()

SRCS(
    config.go
)

IF (OPENSOURCE)
    SRCS(
        endpointset_resolver_stub.go
    )
ELSE()
    SRCS(
        endpointset_resolver_yandex.go
    )
ENDIF()

END()
