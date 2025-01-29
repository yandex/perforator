GO_LIBRARY()

IF(OPENSOURCE)
SRCS(
    interceptor_stub.go
    creds_stub.go
)
ELSE()
SRCS(
    interceptor_yandex.go
    creds_yandex.go
)
ENDIF()

END()
