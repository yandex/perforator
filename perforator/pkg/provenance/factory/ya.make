GO_LIBRARY()

IF (OPENSOURCE)
    SRCS(
        factory.go
    )
ELSE()
    SRCS(
        factory_yandex.go
    )
ENDIF()

END()
