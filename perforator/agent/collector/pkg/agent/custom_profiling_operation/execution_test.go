package custom_profiling_operation

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/yandex/perforator/library/go/core/metrics/nop"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/agent/custom_profiling_operation/mocks"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/agent/custom_profiling_operation/models"
	"github.com/yandex/perforator/perforator/pkg/xlog"
	cpo_proto "github.com/yandex/perforator/perforator/proto/custom_profiling_operation"
	"github.com/yandex/perforator/perforator/proto/lib/time_interval"
)

// mockReporter is a mock implementation of OperationReporter for testing
type mockReporter struct {
	statusUpdates []*models.OperationStatus
	mu            sync.Mutex
	updateChan    chan *models.OperationStatus
}

func newMockReporter() *mockReporter {
	return &mockReporter{
		statusUpdates: make([]*models.OperationStatus, 0),
		updateChan:    make(chan *models.OperationStatus, 10),
	}
}

func (m *mockReporter) UpdateOperationStatus(ctx context.Context, id models.OperationID, status *models.OperationStatus) error {
	m.mu.Lock()
	m.statusUpdates = append(m.statusUpdates, status)
	m.mu.Unlock()

	select {
	case m.updateChan <- status:
	default:
	}

	return nil
}

func (m *mockReporter) waitForStatus(timeout time.Duration) (*models.OperationStatus, bool) {
	select {
	case status := <-m.updateChan:
		return status, true
	case <-time.After(timeout):
		return nil, false
	}
}

func (m *mockReporter) getStatuses() []*models.OperationStatus {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]*models.OperationStatus, len(m.statusUpdates))
	copy(result, m.statusUpdates)
	return result
}

func waitForChan(t *testing.T, done <-chan struct{}, timeout time.Duration) {
	select {
	case <-done:
	case <-time.After(timeout):
		t.Fatal("Waiting did not complete within timeout")
	}
}

func TestNewTimeBoundedOperationExecution_ExpiredInterval(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockController := mocks.NewMockOperationController(ctrl)
	reporter := newMockReporter()
	logger := xlog.ForTest(t)
	metricsRegistry := &nop.Registry{}

	now := time.Now()
	timeInterval := &time_interval.TimeInterval{
		From: timestamppb.New(now.Add(-10 * time.Second)),
		To:   timestamppb.New(now.Add(-1 * time.Second)),
	}

	execution, err := newOperationExecution(logger, metricsRegistry, "test-operation-id", mockController, reporter, timeInterval)

	assert.Error(t, err)
	assert.Nil(t, execution)
	assert.Contains(t, err.Error(), "operation time interval has expired")
}

func TestTimeBoundedOperationExecution_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockController := mocks.NewMockOperationController(ctrl)
	reporter := newMockReporter()
	logger := xlog.ForTest(t)
	metricsRegistry := &nop.Registry{}

	now := time.Now()
	timeInterval := &time_interval.TimeInterval{
		From: timestamppb.New(now.Add(50 * time.Millisecond)),
		To:   timestamppb.New(now.Add(2 * time.Second)),
	}

	mockController.EXPECT().Start(gomock.Any()).Return(nil).Times(1)
	mockController.EXPECT().Stop().Return(nil).Times(1)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	execution, err := newOperationExecution(logger, metricsRegistry, "test-operation-id", mockController, reporter, timeInterval)
	require.NoError(t, err)

	done := make(chan struct{})
	go func() {
		execution.Run(ctx)
		close(done)
	}()

	// Wait for initial Prepared status
	status, ok := reporter.waitForStatus(100 * time.Millisecond)
	require.True(t, ok, "Expected Prepared status")
	assert.Equal(t, cpo_proto.OperationState_Prepared, status.State)

	// Wait for Running status
	status, ok = reporter.waitForStatus(200 * time.Millisecond)
	require.True(t, ok, "Expected Running status to be reported")
	assert.Equal(t, cpo_proto.OperationState_Running, status.State)
	assert.Empty(t, status.Error)

	// Wait for Finished status
	status, ok = reporter.waitForStatus(2 * time.Second)
	require.True(t, ok, "Expected Finished status to be reported")
	assert.Equal(t, cpo_proto.OperationState_Finished, status.State)
	assert.Empty(t, status.Error)

	waitForChan(t, done, time.Second)
}

func TestTimeBoundedOperationExecution_StartOperation_Failure(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockController := mocks.NewMockOperationController(ctrl)
	reporter := newMockReporter()
	logger := xlog.ForTest(t)
	metricsRegistry := &nop.Registry{}

	now := time.Now()
	timeInterval := &time_interval.TimeInterval{
		From: timestamppb.New(now.Add(50 * time.Millisecond)),
		To:   timestamppb.New(now.Add(5 * time.Second)),
	}

	expectedError := errors.New("start failed")
	mockController.EXPECT().Start(gomock.Any()).Return(expectedError).Times(1)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	execution, err := newOperationExecution(logger, metricsRegistry, "test-operation-id", mockController, reporter, timeInterval)
	require.NoError(t, err)

	done := make(chan struct{})
	go func() {
		execution.Run(ctx)
		close(done)
	}()

	// Wait for initial Prepared status
	status, ok := reporter.waitForStatus(10 * time.Millisecond)
	require.True(t, ok, "Expected Prepared status")
	assert.Equal(t, cpo_proto.OperationState_Prepared, status.State)

	// Wait for Failed status
	status, ok = reporter.waitForStatus(100 * time.Millisecond)
	require.True(t, ok, "Expected Failed status to be reported")
	assert.Equal(t, cpo_proto.OperationState_Failed, status.State)
	assert.Equal(t, expectedError.Error(), status.Error)

	waitForChan(t, done, time.Second)
}

func TestTimeBoundedOperationExecution_ImmediateStartAndCancel(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockController := mocks.NewMockOperationController(ctrl)
	reporter := newMockReporter()
	logger := xlog.ForTest(t)
	metricsRegistry := &nop.Registry{}

	now := time.Now()
	// Start time is in the past, so should start immediately
	timeInterval := &time_interval.TimeInterval{
		From: timestamppb.New(now.Add(-1 * time.Second)),
		To:   timestamppb.New(now.Add(5 * time.Second)),
	}

	mockController.EXPECT().Start(gomock.Any()).Return(nil).Times(1)
	mockController.EXPECT().Stop().Return(nil).Times(1)

	ctx, cancel := context.WithCancel(context.Background())

	execution, err := newOperationExecution(logger, metricsRegistry, "test-operation-id", mockController, reporter, timeInterval)
	require.NoError(t, err)

	done := make(chan struct{})
	go func() {
		execution.Run(ctx)
		close(done)
	}()

	// Wait for initial Prepared status
	status, ok := reporter.waitForStatus(10 * time.Millisecond)
	require.True(t, ok, "Expected Prepared status")
	assert.Equal(t, cpo_proto.OperationState_Prepared, status.State)

	// Should receive Running status very quickly (almost immediately)
	status, ok = reporter.waitForStatus(10 * time.Millisecond)
	require.True(t, ok, "Expected Running status to be reported immediately")
	assert.Equal(t, cpo_proto.OperationState_Running, status.State)

	cancel()
	waitForChan(t, done, time.Second)
}
