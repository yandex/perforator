GO_LIBRARY()

PEERDIR(
    ${GOSTD}/io
    ${GOSTD}/time
    perforator/pkg/storage/binary
    perforator/pkg/storage/binary/meta
    perforator/pkg/storage/storage
    perforator/pkg/storage/util
)

GO_MOCKGEN_FROM(perforator/pkg/storage/binary)
GO_MOCKGEN_SOURCE(models.go)
GO_MOCKGEN_PACKAGE(mock)

END()
