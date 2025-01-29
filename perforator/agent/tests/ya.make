# yo ignore:file
RECURSE(
    dummies
    python
)

IF (NOT OPENSOURCE)
    RECURSE(
        yandex-specific
    )
ENDIF()
