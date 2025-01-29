GO_PROGRAM(web)

PEERDIR(perforator/ui-union)

RESOURCE(
    ${ARCADIA_BUILD_ROOT}/perforator/ui-union/output.tar ui.tar
)

SRCS(
    main.go
)

END()
