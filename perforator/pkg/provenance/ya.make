GO_LIBRARY()

SRCS(
    provenance.go
)

END()

RECURSE(
    factory
)

IF (NOT OPENSOURCE)
    RECURSE(
        yandex-specific/arcadia
    )
ENDIF()
