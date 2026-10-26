GO_LIBRARY()

SRCS(
    client.go
    default_endpoint.go
    endpoint.go
    tls.go
    useragent.go
)

IF (NOT OPENSOURCE)
    SRCS(
        yandex_specific.go
    )
ENDIF()

GO_TEST_SRCS(
    endpoint_test.go
)

END()

RECURSE(
    gotest
)
