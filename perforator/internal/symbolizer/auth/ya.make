GO_LIBRARY()

SRCS(
    context.go
    provider.go
)

END()

RECURSE(
    nopauth
)

IF (NOT OPENSOURCE)
    RECURSE(
        yandexoauth
    )
ENDIF()
