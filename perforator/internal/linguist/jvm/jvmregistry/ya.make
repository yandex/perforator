GO_LIBRARY()

PEERDIR(perforator/internal/linguist/jvm/cheatsheets)

SRCS(
    registry.go
    synthesis.go
)

IF(OPENSOURCE)
    SRCS(helper.go)
ELSE()
    SRCS(helper_yandex.go)
ENDIF()

END()
