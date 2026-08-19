package custom_profiling_operation

import (
	"context"
	"errors"
	"sync"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/library/go/core/metrics"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/agent/custom_profiling_operation/models"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/profiler"
	cpo_storage_models "github.com/yandex/perforator/perforator/pkg/storage/custom_profiling_operation"
	"github.com/yandex/perforator/perforator/pkg/xlog"
	cpo_proto "github.com/yandex/perforator/perforator/proto/custom_profiling_operation"
)

var (
	_ models.OperationExecutionRegistry = (*registry)(nil)
)

type execution struct {
	models.OperationExecution
	executionCtx       context.Context
	cancelExecutionCtx context.CancelFunc
}

type registry struct {
	l            xlog.Logger
	profiler     *profiler.Profiler
	reporter     models.OperationReporter
	reg          metrics.Registry
	mutex        sync.Mutex
	executionsWG sync.WaitGroup
	executions   map[models.OperationID]*execution
}

func NewOperationExecutionRegistry(
	l xlog.Logger,
	reg metrics.Registry,
	profiler *profiler.Profiler,
	reporter models.OperationReporter,
) *registry {
	return &registry{
		l:          l,
		profiler:   profiler,
		executions: make(map[models.OperationID]*execution),
		reporter:   reporter,
		reg:        reg,
	}
}

func (r *registry) createOperationExecution(
	l xlog.Logger,
	operation *cpo_proto.Operation,
) (models.OperationExecution, error) {
	if operation.Status != nil && cpo_storage_models.IsTerminalState(operation.Status.State) {
		return nil, errors.New("operation is already in a terminal state")
	}

	controller, err := newOperationController(l, r.profiler, operation.ID, operation.Spec)
	if err != nil {
		return nil, err
	}

	execution, err := newOperationExecution(l, r.reg, operation.ID, controller, r.reporter, operation.Spec.TimeInterval)
	if err != nil {
		return nil, err
	}

	return execution, nil
}

func (r *registry) releaseExecution(id models.OperationID) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	delete(r.executions, id)
}

// Wait blocks until all operation executions created by the registry finish
// their cancellation teardown. Call it only after all calls to Ensure have
// completed and no new calls can start.
func (r *registry) Wait() {
	r.executionsWG.Wait()
}

func (r *registry) Ensure(ctx context.Context, operation *cpo_proto.Operation) (cancelCtx context.CancelFunc, err error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	logger := r.l.With(log.String("operation_id", string(operation.ID)))

	currentExecution, ok := r.executions[operation.ID]
	if ok {
		return currentExecution.cancelExecutionCtx, nil
	}

	operationExecution, err := r.createOperationExecution(logger, operation)
	if err != nil {
		return nil, err
	}

	id := operation.ID

	executionCtx, executionCtxCancel := context.WithCancel(ctx)
	r.executions[id] = &execution{
		OperationExecution: operationExecution,
		executionCtx:       executionCtx,
		cancelExecutionCtx: executionCtxCancel,
	}
	r.executionsWG.Add(1)
	go func() {
		defer r.executionsWG.Done()
		operationExecution.Run(executionCtx)
		r.releaseExecution(id)
		logger.Info(ctx, "Removed operation execution from registry")
	}()

	return executionCtxCancel, nil
}
