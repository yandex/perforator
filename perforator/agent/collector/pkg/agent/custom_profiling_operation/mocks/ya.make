GO_LIBRARY()

PEERDIR(
    perforator/agent/collector/pkg/agent/custom_profiling_operation/models
    perforator/proto/custom_profiling_operation
)

GO_MOCKGEN_FROM(perforator/agent/collector/pkg/agent/custom_profiling_operation/models)
GO_MOCKGEN_SOURCE(models.go)

END()
