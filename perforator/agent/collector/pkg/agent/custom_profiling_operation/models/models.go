package models

import (
	"context"

	cpo_proto "github.com/yandex/perforator/perforator/proto/custom_profiling_operation"
)

// CPO is a short name for Custom Profiling Operation.

type OperationID = string
type OperationSpec = cpo_proto.OperationSpec
type OperationStats = cpo_proto.OperationStats
type OperationStatus = cpo_proto.OperationStatus
type OperationTargetState = cpo_proto.OperationTargetState

// OperationReporter updates operation status
type OperationReporter interface {
	UpdateOperationStatus(ctx context.Context, id OperationID, status *OperationStatus) error
}

// OperationController controls operation state
type OperationController interface {
	Start(ctx context.Context) error
	Stop() error
}

// OperationExecution executes operation
// Execution can be stopped by cancelling the context.
type OperationExecution interface {
	Run(ctx context.Context)
}

// OperationExecutionRegistry is responsible for creating and releasing operation executions
type OperationExecutionRegistry interface {
	// Ensure creates a new operation execution and returns a cancel function that can be used to stop the execution
	Ensure(ctx context.Context, operation *cpo_proto.Operation) (cancelCtx context.CancelFunc, err error)
}

// Handler is responsible for handling operations received from the agent gateway
type Handler interface {
	Handle(ctx context.Context, operation *cpo_proto.Operation) error
}
