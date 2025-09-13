package custom_profiling_operation

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/library/go/core/metrics"
	"github.com/yandex/perforator/library/go/ptr"
	"github.com/yandex/perforator/perforator/pkg/storage/custom_profiling_operation"
	"github.com/yandex/perforator/perforator/pkg/xlog"
	cpo_proto "github.com/yandex/perforator/perforator/proto/custom_profiling_operation"
)

type serviceMetrics struct {
	successfulCollections      metrics.Counter
	failedCollections          metrics.Counter
	podScopedOperationsNumber  metrics.IntGauge
	nodeScopedOperationsNumber metrics.IntGauge
	currentSnapshotAge         metrics.FuncIntGauge
}

type Service struct {
	l       xlog.Logger
	conf    *ServiceConfig
	reg     metrics.Registry
	metrics serviceMetrics

	storage custom_profiling_operation.Storage

	currentSnapshotManager *operationsSnapshotManager
}

func NewService(
	l xlog.Logger,
	reg metrics.Registry,
	conf *ServiceConfig,
	storage custom_profiling_operation.Storage,
) (*Service, error) {
	reg = reg.WithPrefix("custom_profiling_operation_service")

	snapshotManager := newOperationsSnapshotManager()

	return &Service{
		l:    l,
		conf: conf,
		reg:  reg,
		metrics: serviceMetrics{
			successfulCollections:      reg.Counter("background_collection.successful"),
			failedCollections:          reg.Counter("background_collection.failed"),
			podScopedOperationsNumber:  reg.WithTags(map[string]string{"scope": "pod"}).IntGauge("current_operations.gauge"),
			nodeScopedOperationsNumber: reg.WithTags(map[string]string{"scope": "node"}).IntGauge("current_operations.gauge"),
			currentSnapshotAge: reg.FuncIntGauge("current_snapshot_age", func() int64 {
				return int64(time.Since(snapshotManager.getSnapshotTimestamp()).Seconds())
			}),
		},
		storage:                storage,
		currentSnapshotManager: snapshotManager,
	}, nil
}

func (s *Service) collectOperations(ctx context.Context) ([]*cpo_proto.Operation, error) {
	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(ctx, s.conf.CollectOperationsTimeout)
	defer cancel()

	operations, err := s.storage.ListOperations(
		ctx,
		&custom_profiling_operation.OperationFilter{
			StartsBefore: ptr.Time(time.Now().Add(s.conf.PrefetchInterval)),
			EndsAfter:    ptr.Time(time.Now()),
		},
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list operations: %w", err)
	}

	return operations, nil
}

func (s *Service) tryUpdateSnapshot(ctx context.Context) {
	currentOperations, err := s.collectOperations(ctx)
	if err != nil {
		s.metrics.failedCollections.Inc()
		s.l.Warn(ctx, "Failed to collect current operations", log.Error(err))
		return
	}

	s.metrics.successfulCollections.Inc()

	snapshot := buildCustomProfilingOperationSnapshot(currentOperations)

	s.metrics.podScopedOperationsNumber.Set(int64(len(snapshot.podToOperations)))
	s.metrics.nodeScopedOperationsNumber.Set(int64(len(snapshot.nodeToOperations)))

	s.currentSnapshotManager.updateSnapshot(snapshot)

	s.l.Info(ctx, "Updated current custom profiling operations snapshot", log.Int("podToOperations.size", len(snapshot.podToOperations)), log.Int("nodeToOperations.size", len(snapshot.nodeToOperations)))
}

func (s *Service) runSnapshotUpdater(ctx context.Context) error {
	ticker := time.NewTicker(s.conf.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			s.tryUpdateSnapshot(ctx)
		}
	}
}

func (s *Service) Run(ctx context.Context) error {
	err := s.runSnapshotUpdater(ctx)
	if err != nil {
		s.l.Error(ctx, "Stopped current operations snapshot updater due to error", log.Error(err))
	}

	return err
}

func (s *Service) longPollOperations(ctx context.Context, req *cpo_proto.PollOperationsRequest) (*cpo_proto.PollOperationsResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, s.conf.LongPollingTimeout)
	defer cancel()

	watcher := newLongPollRequestSnapshotWatcher(s.currentSnapshotManager)
	return watcher.longPollOperations(ctx, req), nil
}

// implements CustomProfilingOperationService/PollOperations
func (s *Service) PollOperations(ctx context.Context, req *cpo_proto.PollOperationsRequest) (*cpo_proto.PollOperationsResponse, error) {
	if req == nil || req.Filter == nil {
		return nil, status.Errorf(codes.InvalidArgument, "req is nil or req.Filter is nil")
	}

	if s.currentSnapshotManager.getSnapshot() == nil {
		return nil, status.Errorf(codes.Internal, "current operations snapshot is nil")
	}

	// In case of unset LongPollingData we will use fast path of long polling mechanism
	return s.longPollOperations(ctx, req)
}

// implements CustomProfilingOperationService/UpdateOperationStatus
func (s *Service) UpdateOperationStatus(ctx context.Context, req *cpo_proto.UpdateOperationStatusRequest) (*cpo_proto.UpdateOperationStatusResponse, error) {
	if req == nil || req.ID == "" {
		return nil, status.Errorf(codes.InvalidArgument, "req is nil or req.ID is empty")
	}

	err := s.storage.UpdateOperationStatus(ctx, custom_profiling_operation.OperationID(req.ID), req.Status)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update operation status: %v", err)
	}

	return &cpo_proto.UpdateOperationStatusResponse{}, nil
}
