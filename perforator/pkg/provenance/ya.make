GO_LIBRARY()

SRCS(
    provenance.go
)

END()

IF (NOT OPENSOURCE)
    RECURSE(
        yandex-specific/arcadia
    )
ENDIF()
