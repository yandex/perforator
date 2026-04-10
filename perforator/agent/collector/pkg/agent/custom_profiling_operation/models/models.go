package models

import (
	"context"

	cpo_proto "github.com/yandex/perforator/perforator/proto/custom_profiling_operation"
)

// ResourceLeakError indicates that some OS or internal resources (like BPF programs or perf events)
// could not be released during the operation stop or start rollback. The operation should be considered a zombie.
type ResourceLeakError interface {
	error
	IsResourceLeak() bool
}

// ProfilingDataLostError indicates that some or all of the collected profiling data could be lost
type ProfilingDataLostError interface {
	error
	IsProfilingDataLost() bool
}

type resourceLeakError struct {
	err error
}

func (e *resourceLeakError) Error() string        { return e.err.Error() }
func (e *resourceLeakError) Unwrap() error        { return e.err }
func (e *resourceLeakError) IsResourceLeak() bool { return true }

// NewResourceLeakError wraps an existing error with the ResourceLeakError marker.
func NewResourceLeakError(err error) error {
	if err == nil {
		return nil
	}
	return &resourceLeakError{err: err}
}

type profilingDataLostError struct {
	err error
}

func (e *profilingDataLostError) Error() string             { return e.err.Error() }
func (e *profilingDataLostError) Unwrap() error             { return e.err }
func (e *profilingDataLostError) IsProfilingDataLost() bool { return true }

// NewProfilingDataLostError wraps an existing error with the ProfilingDataLostError marker.
func NewProfilingDataLostError(err error) error {
	if err == nil {
		return nil
	}
	return &profilingDataLostError{err: err}
}

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
	// Start attempts to initialize and start the operation.
	// It guarantees an all-or-nothing behavior: if any initialization step fails,
	// it attempts to rollback all partially allocated resources.
	//
	// If the returned error contains ResourceLeakError, it means the initialization failed
	// AND the subsequent rollback also failed, leaving the system with leaked resources.
	Start(ctx context.Context) error

	// Stop attempts to gracefully stop the operation and release all associated resources.
	// It guarantees that a best-effort cleanup is performed for all resources.
	//
	// If the returned error contains ResourceLeakError, it means some OS or internal resources
	// (like BPF programs or uprobes) could not be closed, and the operation is considered a zombie.
	// If the returned error contains ProfilingDataLostError or context cancellation/timeout,
	// it means resources were freed successfully, but some or all collected data could not be saved.
	Stop(ctx context.Context) error
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
