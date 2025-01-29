GO_LIBRARY()

SRCS(
    client.go
    tls.go
    useragent.go
)

IF (OPENSOURCE)
    SRCS(
        default_endpoint.go
    )
ELSE()
    SRCS(
        default_endpoint_yandex.go
    )
ENDIF()



END()
