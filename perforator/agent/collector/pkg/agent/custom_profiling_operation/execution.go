package custom_profiling_operation

import (
	"context"
	"errors"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/library/go/core/metrics"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/agent/custom_profiling_operation/models"
	cpo_models "github.com/yandex/perforator/perforator/pkg/storage/custom_profiling_operation"
	"github.com/yandex/perforator/perforator/pkg/xlog"
	cpo_proto "github.com/yandex/perforator/perforator/proto/custom_profiling_operation"
	"github.com/yandex/perforator/perforator/proto/lib/time_interval"
)

var (
	_ models.OperationExecution = (*operationExecution)(nil)
)

var (
	errFinished = errors.New("finished")
)

const (
	defaultStatusOutputChannelSize = 10
)

type operationExecutionMetrics struct {
	prepared metrics.Counter
	started  metrics.Counter
	failed   metrics.Counter
	stopped  metrics.Counter
	finished metrics.Counter
	zombie   metrics.Counter
}

type operationExecution struct {
	l            xlog.Logger
	id           models.OperationID
	reporter     models.OperationReporter
	timeInterval *time_interval.TimeInterval

	// This mutex is used to protect concurrent access to operation and status
	mutex               sync.Mutex
	operationController models.OperationController
	status              *models.OperationStatus

	stopOnce sync.Once

	statusChan chan *models.OperationStatus

	metrics operationExecutionMetrics
}

func newOperationExecution(
	l xlog.Logger,
	reg metrics.Registry,
	id models.OperationID,
	operationController models.OperationController,
	reporter models.OperationReporter,
	timeInterval *time_interval.TimeInterval,
) (*operationExecution, error) {
	if timeInterval.To.AsTime().Before(time.Now()) {
		return nil, errors.New("operation time interval has expired")
	}

	reg = reg.WithPrefix("custom_profiling_operation")

	execution := &operationExecution{
		l:                   l,
		id:                  id,
		operationController: operationController,
		reporter:            reporter,
		timeInterval:        timeInterval,
		statusChan:          make(chan *models.OperationStatus, defaultStatusOutputChannelSize),
		status: &models.OperationStatus{
			State:     cpo_proto.OperationState_Prepared,
			Timestamp: timestamppb.Now(),
		},
		metrics: operationExecutionMetrics{
			prepared: reg.Counter("prepared.count"),
			started:  reg.Counter("started.count"),
			failed:   reg.Counter("failed.count"),
			stopped:  reg.Counter("stopped.count"),
			finished: reg.Counter("finished.count"),
			zombie:   reg.Counter("zombie.count"),
		},
	}
	execution.sendStatusSafe()

	return execution, nil
}

// sendStatusSafe sends a clone of the current status to the channel to avoid sharing the same pointer.
// It the statusChan is closed, the status is not sent.
func (e *operationExecution) sendStatusSafe() {
	if e.statusChan == nil {
		return
	}

	statusCopy := proto.Clone(e.status).(*models.OperationStatus)
	e.statusChan <- statusCopy
}

func (e *operationExecution) startOperation(ctx context.Context) (started bool) {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	if e.status.State != cpo_proto.OperationState_Prepared {
		return false
	}

	e.l.Info(ctx, "Starting CPO")
	err := e.operationController.Start(ctx)
	if err != nil {
		e.l.Error(ctx, "Failed to start CPO", log.Error(err))
		e.failOperation(err)
		e.metrics.failed.Inc()
		return false
	}

	e.l.Info(ctx, "Successfully started CPO")
	e.metrics.started.Inc()
	e.updateState(cpo_proto.OperationState_Running)
	return true
}

func (e *operationExecution) stopOperationImpl(ctx context.Context, finalState cpo_proto.OperationState) {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	if e.status.State != cpo_proto.OperationState_Running {
		return
	}

	e.l.Info(ctx, "Stopping CPO")
	err := e.operationController.Stop()
	if err != nil {
		e.l.Error(ctx, "Failed to stop CPO", log.Error(err))
		// This should never occur - set up monitoring for this
		e.metrics.zombie.Inc()
		return
	}

	e.l.Info(ctx, "Successfully stopped CPO")
	e.updateState(finalState)

	if finalState == cpo_proto.OperationState_Finished {
		e.metrics.finished.Inc()
	} else if finalState == cpo_proto.OperationState_Stopped {
		e.metrics.stopped.Inc()
	}
}

func (e *operationExecution) finishOperation(ctx context.Context) {
	e.stopOnce.Do(func() {
		e.stopOperationImpl(ctx, cpo_proto.OperationState_Finished)
	})
}

func (e *operationExecution) stopOperation(ctx context.Context) {
	e.stopOnce.Do(func() {
		e.stopOperationImpl(ctx, cpo_proto.OperationState_Stopped)
	})
}

func (e *operationExecution) runReportStatus(ctx context.Context) {
	for status := range e.statusChan {
		err := e.reporter.UpdateOperationStatus(ctx, e.id, status)
		if err != nil {
			e.l.Error(ctx, "Failed to report status", log.Error(err))
		}
	}
}

func (e *operationExecution) Run(ctx context.Context) {
	e.metrics.prepared.Inc()

	runDuration := time.Until(e.timeInterval.To.AsTime())
	if runDuration < 0 {
		runDuration = 0
	}

	// Create context with cause for normal finish
	limitedCtx, cancel := context.WithTimeoutCause(ctx, runDuration, errFinished)
	defer cancel()

	g, gCtx := errgroup.WithContext(limitedCtx)

	g.Go(func() error {
		durationBeforeStart := time.Until(e.timeInterval.From.AsTime())
		if durationBeforeStart > 0 {
			select {
			case <-time.After(durationBeforeStart):
			case <-gCtx.Done():
				return nil
			}
		}

		started := e.startOperation(gCtx)
		if !started {
			return nil
		}

		<-gCtx.Done()

		cause := context.Cause(gCtx)
		if errors.Is(cause, errFinished) {
			e.finishOperation(gCtx)
		} else {
			e.stopOperation(gCtx)
		}

		return nil
	})

	g.Go(func() error {
		e.runReportStatus(ctx)
		return nil
	})

	_ = g.Wait()
}

func (e *operationExecution) closeStatusChan() {
	close(e.statusChan)
	e.statusChan = nil
}

func (e *operationExecution) maybeCloseStatusChan() {
	if cpo_models.IsTerminalState(e.status.State) {
		e.closeStatusChan()
	}
}

func (e *operationExecution) failOperation(operationError error) {
	e.status.Error = operationError.Error()
	e.status.Timestamp = timestamppb.Now()
	e.status.State = cpo_proto.OperationState_Failed
	e.sendStatusSafe()
	e.closeStatusChan()
}

func (e *operationExecution) updateState(state cpo_proto.OperationState) {
	e.status.State = state
	e.status.Timestamp = timestamppb.Now()
	e.sendStatusSafe()
	e.maybeCloseStatusChan()
}
