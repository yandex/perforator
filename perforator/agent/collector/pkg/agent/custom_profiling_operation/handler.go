package custom_profiling_operation

import (
	"context"

	"github.com/yandex/perforator/perforator/agent/collector/pkg/agent/custom_profiling_operation/models"
	"github.com/yandex/perforator/perforator/pkg/xlog"
	cpo_proto "github.com/yandex/perforator/perforator/proto/custom_profiling_operation"
)

var (
	_ models.Handler = (*cpoHandler)(nil)
)

type cpoHandler struct {
	l        xlog.Logger
	registry models.OperationExecutionRegistry
}

func NewCPOHandler(l xlog.Logger, registry models.OperationExecutionRegistry) *cpoHandler {
	return &cpoHandler{
		l:        l,
		registry: registry,
	}
}

func (h *cpoHandler) Handle(ctx context.Context, operation *cpo_proto.Operation) error {
	cancelExecutionCtx, err := h.registry.Ensure(ctx, operation)
	if err != nil {
		return err
	}

	if operation.TargetState != nil && operation.TargetState.State == cpo_proto.OperationState_Stopped {
		cancelExecutionCtx()
	}

	return nil
}
