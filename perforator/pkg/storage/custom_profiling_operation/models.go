package custom_profiling_operation

import (
	"context"
	"time"

	"github.com/yandex/perforator/perforator/pkg/storage/util"
	cpo_proto "github.com/yandex/perforator/perforator/proto/custom_profiling_operation"
)

type CustomProfilingOperationStorageType string

const (
	Postgres CustomProfilingOperationStorageType = "postgres"
)

type Storage interface {
	// Returns operation if it already exists, otherwise creates a new one.
	InsertOperation(ctx context.Context, id OperationID, spec *cpo_proto.OperationSpec) (*cpo_proto.Operation, error)

	GetOperation(ctx context.Context, id OperationID) (*cpo_proto.Operation, error)

	StopOperation(ctx context.Context, id OperationID) error

	UpdateOperationStatus(
		ctx context.Context,
		id OperationID,
		newStatus *cpo_proto.OperationStatus,
	) error

	ListOperations(ctx context.Context, filter *OperationFilter, pagination *util.Pagination) ([]*cpo_proto.Operation, error)
}

type OperationID string

type OperationFilter struct {
	EndsAfter    *time.Time
	StartsBefore *time.Time
}

func IsTerminalState(state cpo_proto.OperationState) bool {
	return state == cpo_proto.OperationState_Finished || state == cpo_proto.OperationState_Failed || state == cpo_proto.OperationState_Stopped
}

var (
	stateOrder = map[cpo_proto.OperationState]int{
		cpo_proto.OperationState_Unknown:  0,
		cpo_proto.OperationState_Prepared: 1,
		cpo_proto.OperationState_Running:  2,
		cpo_proto.OperationState_Failed:   3,
		cpo_proto.OperationState_Finished: 3,
		cpo_proto.OperationState_Stopped:  3,
	}
)

func IsAllowedStateChange(oldState, newState cpo_proto.OperationState) bool {
	return stateOrder[oldState] < stateOrder[newState]
}
