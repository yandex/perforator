GO_LIBRARY()

SRCS(
    options.go
    registry.go
)

IF (OPENSOURCE)
    SRCS(
        metrics.go
    )
ELSE()
    SRCS(
        metrics_yandex.go
    )
ENDIF()

END()
