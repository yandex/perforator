GO_LIBRARY()

SRCS(
    tlsconfig.go
    utils.go
)

IF (OPENSOURCE)
    SRCS(
        systemcert.go
    )
ELSE()
    SRCS(
        certifi_yandex.go
    )
ENDIF()

END()
