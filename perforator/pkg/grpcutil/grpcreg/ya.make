GO_LIBRARY()

SRCS(
    grpcreg.go
)

IF (OPENSOURCE)
    SRCS(
        grpcreg_stub.go
    )
ELSE()
    SRCS(
        grpcreg_yandex.go
    )
ENDIF()

END()
