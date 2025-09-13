GO_LIBRARY()

PEERDIR(
    perforator/pkg/storage/custom_profiling_operation
    perforator/pkg/storage/util
    perforator/proto/custom_profiling_operation
)

GO_MOCKGEN_FROM(perforator/pkg/storage/custom_profiling_operation)
GO_MOCKGEN_SOURCE(models.go)

END()
