GO_LIBRARY()

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
