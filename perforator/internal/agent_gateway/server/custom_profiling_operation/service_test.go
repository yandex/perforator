package custom_profiling_operation

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/yandex/perforator/library/go/core/metrics/nop"
	"github.com/yandex/perforator/perforator/pkg/storage/custom_profiling_operation"
	operation_mocks "github.com/yandex/perforator/perforator/pkg/storage/custom_profiling_operation/mocks"
	"github.com/yandex/perforator/perforator/pkg/storage/util"
	"github.com/yandex/perforator/perforator/pkg/xlog"
	cpo_proto "github.com/yandex/perforator/perforator/proto/custom_profiling_operation"
)

const (
	testTimeout = 5 * time.Second
)

func createTestOperationForNode(id, hostNode string) *cpo_proto.Operation {
	return &cpo_proto.Operation{
		ID: id,
		Spec: &cpo_proto.OperationSpec{
			Target: &cpo_proto.Target{
				Target: &cpo_proto.Target_NodeProcess{
					NodeProcess: &cpo_proto.NodeProcessTarget{
						Host: hostNode,
					},
				},
			},
		},
	}
}

func createTestOperationForPod(id, pod string) *cpo_proto.Operation {
	return &cpo_proto.Operation{
		ID: id,
		Spec: &cpo_proto.OperationSpec{
			Target: &cpo_proto.Target{
				Target: &cpo_proto.Target_Pod{
					Pod: &cpo_proto.PodTarget{
						Pod: pod,
					},
				},
			},
		},
	}
}

func createTestService(
	t *testing.T,
	operationsMockStorage *operation_mocks.MockStorage,
	longPollingTimeout time.Duration,
) *Service {
	logger := xlog.ForTest(t)
	reg := nop.Registry{}

	conf := &ServiceConfig{
		PollInterval:       100 * time.Millisecond,
		PrefetchInterval:   5 * time.Minute,
		LongPollingTimeout: longPollingTimeout,
	}

	service, err := NewService(logger, reg, conf, operationsMockStorage)
	require.NoError(t, err)
	return service
}

func TestService_PollOperations_Immediate(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	operationsMockStorage := operation_mocks.NewMockStorage(ctrl)
	service := createTestService(t, operationsMockStorage, 2*time.Second)

	testOperations := []*cpo_proto.Operation{
		createTestOperationForNode("op-node1-1", "node1"),
		createTestOperationForNode("op-node1-2", "node1"),
		createTestOperationForPod("op-pod1", "pod1"),
		createTestOperationForPod("op-pod2", "pod2"),
		createTestOperationForPod("op-pod3", "pod3"),
	}

	operationsMockStorage.EXPECT().
		ListOperations(gomock.Any(), gomock.Any(), gomock.Eq((*util.Pagination)(nil))).
		Return(testOperations, nil).AnyTimes()

	service.tryUpdateSnapshot(ctx)

	tests := []struct {
		name          string
		host          string
		pods          []string
		expectedCount int
		expectedIDs   []string
	}{
		{
			name:          "node1_only",
			host:          "node1",
			pods:          nil,
			expectedCount: 2,
			expectedIDs:   []string{"op-node1-1", "op-node1-2"},
		},
		{
			name:          "pods_only",
			host:          "",
			pods:          []string{"pod1", "pod2"},
			expectedCount: 2,
			expectedIDs:   []string{"op-pod1", "op-pod2"},
		},
		{
			name:          "node_and_pods",
			host:          "node1",
			pods:          []string{"pod1"},
			expectedCount: 3,
			expectedIDs:   []string{"op-node1-1", "op-node1-2", "op-pod1"},
		},
		{
			name:          "empty_filter",
			host:          "",
			pods:          nil,
			expectedCount: 0,
			expectedIDs:   []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &cpo_proto.PollOperationsRequest{
				Filter: &cpo_proto.PollOperationsFilter{
					Host:                tt.host,
					Pods:                tt.pods,
					MaxPrefetchInterval: durationpb.New(5 * time.Minute),
				},
			}

			resp, err := service.PollOperations(ctx, req)
			require.NoError(t, err)
			require.NotNil(t, resp)

			assert.Len(t, resp.Operations, tt.expectedCount)

			actualIDs := make([]string, len(resp.Operations))
			for i, op := range resp.Operations {
				actualIDs[i] = op.ID
			}
			assert.ElementsMatch(t, tt.expectedIDs, actualIDs)
		})
	}
}

func TestService_PollOperations_ValidationErrors(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	operationsMockStorage := operation_mocks.NewMockStorage(ctrl)
	service := createTestService(t, operationsMockStorage, 2*time.Second)

	testOperations := []*cpo_proto.Operation{
		createTestOperationForNode("test-node-op", "test-node"),
		createTestOperationForPod("test-pod-op", "test-pod"),
	}
	operationsMockStorage.EXPECT().
		ListOperations(gomock.Any(), gomock.Any(), gomock.Eq((*util.Pagination)(nil))).
		Return(testOperations, nil).AnyTimes()

	service.tryUpdateSnapshot(ctx)

	tests := []struct {
		name        string
		req         *cpo_proto.PollOperationsRequest
		expectedErr string
	}{
		{
			name:        "nil_request",
			req:         nil,
			expectedErr: "req is nil or req.Filter is nil",
		},
		{
			name: "nil_filter",
			req: &cpo_proto.PollOperationsRequest{
				Filter: nil,
			},
			expectedErr: "req is nil or req.Filter is nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := service.PollOperations(ctx, tt.req)

			require.Error(t, err)
			assert.Nil(t, resp)

			assert.Contains(t, err.Error(), tt.expectedErr)
			assert.Contains(t, err.Error(), "InvalidArgument")
		})
	}
}

func TestService_PollOperations_SingleClientLongPolling(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	operationsMockStorage := operation_mocks.NewMockStorage(ctrl)
	longPollingTimeout := 2 * time.Second
	service := createTestService(t, operationsMockStorage, longPollingTimeout)

	initialOperations := []*cpo_proto.Operation{
		createTestOperationForNode("op-node2-1", "node2"),
		createTestOperationForPod("op-pod4", "pod4"),
		createTestOperationForPod("op-pod5", "pod5"),
	}

	updatedOperations := []*cpo_proto.Operation{
		createTestOperationForNode("op-node2-1", "node2"),
		createTestOperationForPod("op-pod4", "pod4"),
		createTestOperationForPod("op-pod5", "pod5"),
		// New operation
		createTestOperationForNode("op-node2-2", "node2"),
	}

	operationsMockStorage.EXPECT().
		ListOperations(gomock.Any(), gomock.Any(), gomock.Eq((*util.Pagination)(nil))).
		Return(initialOperations, nil).Times(1)

	service.tryUpdateSnapshot(ctx)

	// First request - immediate polling
	req1 := &cpo_proto.PollOperationsRequest{
		Filter: &cpo_proto.PollOperationsFilter{
			Host:                "node2",
			Pods:                []string{"pod4", "pod5"},
			MaxPrefetchInterval: durationpb.New(5 * time.Minute),
		},
	}

	resp1, err := service.PollOperations(ctx, req1)
	require.NoError(t, err)
	require.NotNil(t, resp1)
	assert.Len(t, resp1.Operations, 3) // node2-1 + pod4 + pod5

	assert.NotNil(t, resp1.NextLongPollingData)
	initialVersion := resp1.NextLongPollingData.GetOperationsVersion()

	operationsMockStorage.EXPECT().
		ListOperations(gomock.Any(), gomock.Any(), gomock.Eq((*util.Pagination)(nil))).
		Return(updatedOperations, nil).AnyTimes()

	// Simulate snapshot update in background
	go func() {
		// Small delay to let long polling start
		time.Sleep(50 * time.Millisecond)
		service.tryUpdateSnapshot(ctx)
	}()

	// Second request - long polling with version from first request
	req2 := &cpo_proto.PollOperationsRequest{
		Filter: &cpo_proto.PollOperationsFilter{
			Host:                "node2",
			Pods:                []string{"pod4", "pod5"},
			MaxPrefetchInterval: durationpb.New(5 * time.Minute),
		},
		LongPollingData: &cpo_proto.LongPollingData{
			Data: &cpo_proto.LongPollingData_OperationsVersion{
				OperationsVersion: initialVersion,
			},
		},
	}

	resp2, err := service.PollOperations(ctx, req2)
	require.NoError(t, err)
	require.NotNil(t, resp2)
	assert.Len(t, resp2.Operations, 4) // node2-1 + pod4 + pod5 + node2-2

	// Version should be different
	newVersion := resp2.NextLongPollingData.GetOperationsVersion()
	assert.NotEqual(t, initialVersion, newVersion)

	actualIDs := make([]string, len(resp2.Operations))
	for i, op := range resp2.Operations {
		actualIDs[i] = op.ID
	}
	expectedIDs := []string{"op-node2-1", "op-pod4", "op-pod5", "op-node2-2"}
	assert.ElementsMatch(t, expectedIDs, actualIDs)

	req3 := &cpo_proto.PollOperationsRequest{
		Filter: &cpo_proto.PollOperationsFilter{
			Host:                "node2",
			Pods:                []string{"pod4", "pod5"},
			MaxPrefetchInterval: durationpb.New(5 * time.Minute),
		},
		LongPollingData: &cpo_proto.LongPollingData{
			Data: &cpo_proto.LongPollingData_OperationsVersion{
				OperationsVersion: newVersion, // Same version as previous response
			},
		},
	}

	startTime := time.Now()
	resp3, err := service.PollOperations(ctx, req3)
	duration := time.Since(startTime)

	require.NoError(t, err)
	require.NotNil(t, resp3)

	assert.GreaterOrEqual(t, duration, longPollingTimeout,
		"Long polling should wait at least %v when no updates, but returned in %v", longPollingTimeout, duration)
	assert.LessOrEqual(t, duration, longPollingTimeout+500*time.Millisecond,
		"Long polling should timeout around %v when no updates, but took %v", longPollingTimeout, duration)

	assert.Len(t, resp3.Operations, 4)
	timeoutVersion := resp3.NextLongPollingData.GetOperationsVersion()
	assert.Equal(t, newVersion, timeoutVersion, "Version should remain the same when no changes occur during timeout")

	t.Logf("Long polling correctly timed out after %v when no updates (expected ~%v)", duration, longPollingTimeout)
}

func TestService_PollOperations_LongPolling_FastPath(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	operationsMockStorage := operation_mocks.NewMockStorage(ctrl)
	longPollingTimeout := 3 * time.Second
	service := createTestService(t, operationsMockStorage, longPollingTimeout)

	initialOperations := []*cpo_proto.Operation{
		createTestOperationForNode("op-node-initial", "test-node"),
		createTestOperationForPod("op-pod-initial", "test-pod"),
	}

	updatedOperations := []*cpo_proto.Operation{
		createTestOperationForNode("op-node-initial", "test-node"),
		createTestOperationForPod("op-pod-initial", "test-pod"),
		// New operations
		createTestOperationForNode("op-node-new", "test-node"),
		createTestOperationForPod("op-pod-new", "test-pod"),
	}

	operationsMockStorage.EXPECT().
		ListOperations(gomock.Any(), gomock.Any(), gomock.Eq((*util.Pagination)(nil))).
		Return(initialOperations, nil).Times(1)

	service.tryUpdateSnapshot(ctx)

	req1 := &cpo_proto.PollOperationsRequest{
		Filter: &cpo_proto.PollOperationsFilter{
			Host:                "test-node",
			Pods:                []string{"test-pod"},
			MaxPrefetchInterval: durationpb.New(5 * time.Minute),
		},
	}

	resp1, err := service.PollOperations(ctx, req1)
	require.NoError(t, err)
	require.NotNil(t, resp1)
	assert.Len(t, resp1.Operations, 2)

	initialVersion := resp1.NextLongPollingData.GetOperationsVersion()
	assert.NotZero(t, initialVersion, "Initial version should not be zero")

	operationsMockStorage.EXPECT().
		ListOperations(gomock.Any(), gomock.Any(), gomock.Eq((*util.Pagination)(nil))).
		Return(updatedOperations, nil).Times(1)

	service.tryUpdateSnapshot(ctx)

	// Make long polling request with old version - should return immediately (fast path)
	req2 := &cpo_proto.PollOperationsRequest{
		Filter: &cpo_proto.PollOperationsFilter{
			Host:                "test-node",
			Pods:                []string{"test-pod"},
			MaxPrefetchInterval: durationpb.New(5 * time.Minute),
		},
		LongPollingData: &cpo_proto.LongPollingData{
			Data: &cpo_proto.LongPollingData_OperationsVersion{
				OperationsVersion: initialVersion, // Old version
			},
		},
	}

	startTime := time.Now()
	resp2, err := service.PollOperations(ctx, req2)
	duration := time.Since(startTime)

	require.NoError(t, err)
	require.NotNil(t, resp2)

	assert.Less(t, duration, 100*time.Millisecond,
		"Fast path should return quickly, but took %v", duration)

	assert.Len(t, resp2.Operations, 4, "Should return all updated operations")

	newVersion := resp2.NextLongPollingData.GetOperationsVersion()
	assert.NotEqual(t, initialVersion, newVersion,
		"Version should be updated after fast path")
	assert.Greater(t, newVersion, initialVersion,
		"New version should be greater than initial version")

	actualIDs := make([]string, len(resp2.Operations))
	for i, op := range resp2.Operations {
		actualIDs[i] = op.ID
	}
	expectedIDs := []string{"op-node-initial", "op-pod-initial", "op-node-new", "op-pod-new"}
	assert.ElementsMatch(t, expectedIDs, actualIDs,
		"Should return all operations including new ones")

	t.Logf("Fast path correctly returned in %v (much faster than timeout %v)",
		duration, longPollingTimeout)
}

func TestService_PollOperations_MultipleAgentsLongPolling(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	operationsMockStorage := operation_mocks.NewMockStorage(ctrl)
	service := createTestService(t, operationsMockStorage, 3*time.Second)

	agentsCount := 10

	type agentClient struct {
		name     string
		nodeID   string
		podNames []string
	}

	agents := make([]agentClient, agentsCount)
	for i := 0; i < agentsCount; i++ {
		agents[i] = agentClient{
			name:   fmt.Sprintf("agent%d", i+1),
			nodeID: fmt.Sprintf("node%d", i+1),
			podNames: []string{
				fmt.Sprintf("pod%d-1", i+1),
				fmt.Sprintf("pod%d-2", i+1),
				fmt.Sprintf("pod%d-3", i+1),
			},
		}
	}

	// Initial operations: each agent has 1 node operation + 2 pod operations (3 total)
	initialOperations := make([]*cpo_proto.Operation, 0)
	for _, agent := range agents {
		initialOperations = append(initialOperations,
			createTestOperationForNode(fmt.Sprintf("op-%s-node", agent.name), agent.nodeID))

		for j := 0; j < 2; j++ {
			initialOperations = append(initialOperations,
				createTestOperationForPod(fmt.Sprintf("op-%s-pod%d", agent.name, j+1), agent.podNames[j]))
		}
	}

	updatedOperations := make([]*cpo_proto.Operation, len(initialOperations))
	copy(updatedOperations, initialOperations)

	for _, agent := range agents {
		updatedOperations = append(updatedOperations,
			createTestOperationForPod(fmt.Sprintf("op-%s-pod3", agent.name), agent.podNames[2]))
	}

	operationsMockStorage.EXPECT().
		ListOperations(gomock.Any(), gomock.Any(), gomock.Eq((*util.Pagination)(nil))).
		Return(initialOperations, nil).Times(1)

	service.tryUpdateSnapshot(ctx)

	// Step 1: All agents get initial state (3 operations each)
	initialVersions := make([]uint64, len(agents))
	for i, agent := range agents {
		req := &cpo_proto.PollOperationsRequest{
			Filter: &cpo_proto.PollOperationsFilter{
				Host:                agent.nodeID,
				Pods:                agent.podNames,
				MaxPrefetchInterval: durationpb.New(5 * time.Minute),
			},
		}

		resp, err := service.PollOperations(ctx, req)
		require.NoError(t, err, "Agent %s initial request failed", agent.name)
		require.NotNil(t, resp)

		assert.Len(t, resp.Operations, 3, "Agent %s should have 3 initial operations", agent.name)

		actualIDs := make([]string, len(resp.Operations))
		for j, op := range resp.Operations {
			actualIDs[j] = op.ID
		}
		expectedIDs := []string{
			fmt.Sprintf("op-%s-node", agent.name),
			fmt.Sprintf("op-%s-pod1", agent.name),
			fmt.Sprintf("op-%s-pod2", agent.name),
		}
		assert.ElementsMatch(t, expectedIDs, actualIDs, "Agent %s got wrong operations", agent.name)

		initialVersions[i] = resp.NextLongPollingData.GetOperationsVersion()
		assert.NotZero(t, initialVersions[i], "Agent %s initial version is zero", agent.name)
	}

	operationsMockStorage.EXPECT().
		ListOperations(gomock.Any(), gomock.Any(), gomock.Eq((*util.Pagination)(nil))).
		Return(updatedOperations, nil).AnyTimes()

	// Step 2: Start long polling for all agents simultaneously
	type agentResult struct {
		agentIndex int
		response   *cpo_proto.PollOperationsResponse
		err        error
	}

	resultChan := make(chan agentResult, len(agents))

	for i, agent := range agents {
		go func(agentIndex int, agent agentClient) {
			req := &cpo_proto.PollOperationsRequest{
				Filter: &cpo_proto.PollOperationsFilter{
					Host:                agent.nodeID,
					Pods:                agent.podNames,
					MaxPrefetchInterval: durationpb.New(5 * time.Minute),
				},
				LongPollingData: &cpo_proto.LongPollingData{
					Data: &cpo_proto.LongPollingData_OperationsVersion{
						OperationsVersion: initialVersions[agentIndex],
					},
				},
			}

			resp, err := service.PollOperations(ctx, req)
			resultChan <- agentResult{
				agentIndex: agentIndex,
				response:   resp,
				err:        err,
			}
		}(i, agent)
	}

	go func() {
		time.Sleep(2 * time.Second)
		service.tryUpdateSnapshot(ctx)
	}()

	// Step 3: Collect results from all agents
	receivedResults := make([]agentResult, 0, len(agents))
	for i := 0; i < len(agents); i++ {
		select {
		case result := <-resultChan:
			receivedResults = append(receivedResults, result)
		case <-time.After(testTimeout):
			t.Fatal("Timeout waiting for long polling responses")
		}
	}

	// Step 4: Verify results for each agent
	for _, result := range receivedResults {
		agent := agents[result.agentIndex]

		require.NoError(t, result.err, "Agent %s long polling failed", agent.name)
		require.NotNil(t, result.response, "Agent %s got nil response", agent.name)

		assert.Len(t, result.response.Operations, 4, "Agent %s should have 4 updated operations", agent.name)

		newVersion := result.response.NextLongPollingData.GetOperationsVersion()
		assert.NotEqual(t, initialVersions[result.agentIndex], newVersion,
			"Agent %s version should have changed", agent.name)

		actualIDs := make([]string, len(result.response.Operations))
		for j, op := range result.response.Operations {
			actualIDs[j] = op.ID
		}
		expectedIDs := []string{
			fmt.Sprintf("op-%s-node", agent.name),
			fmt.Sprintf("op-%s-pod1", agent.name),
			fmt.Sprintf("op-%s-pod2", agent.name),
			fmt.Sprintf("op-%s-pod3", agent.name),
		}
		assert.ElementsMatch(t, expectedIDs, actualIDs, "Agent %s got wrong updated operations", agent.name)

		t.Logf("Agent %s successfully received %d operations: %v",
			agent.name, len(actualIDs), actualIDs)
	}

	assert.Len(t, receivedResults, len(agents), "Not all agents received responses")

	t.Logf("Successfully tested %d agents with long polling", len(agents))
}

func TestService_UpdateOperationStatus_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	operationsMockStorage := operation_mocks.NewMockStorage(ctrl)
	service := createTestService(t, operationsMockStorage, 2*time.Second)

	operationID := "test-operation-id"
	status := &cpo_proto.OperationStatus{
		State:     cpo_proto.OperationState_Running,
		Error:     "",
		Timestamp: timestamppb.Now(),
		Stats: &cpo_proto.OperationStats{
			CollectedProfilesCount: 5,
		},
	}

	operationsMockStorage.EXPECT().
		UpdateOperationStatus(
			gomock.Any(),
			custom_profiling_operation.OperationID(operationID),
			status,
		).
		Return(nil).
		Times(1)

	req := &cpo_proto.UpdateOperationStatusRequest{
		ID:     operationID,
		Status: status,
	}

	resp, err := service.UpdateOperationStatus(ctx, req)

	require.NoError(t, err)
	require.NotNil(t, resp)

	assert.Equal(t, &cpo_proto.UpdateOperationStatusResponse{}, resp)
}

func TestService_UpdateOperationStatus_StorageError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	operationsMockStorage := operation_mocks.NewMockStorage(ctrl)
	service := createTestService(t, operationsMockStorage, 2*time.Second)

	operationID := "test-operation-id"
	status := &cpo_proto.OperationStatus{
		State:     cpo_proto.OperationState_Failed,
		Error:     "test error",
		Timestamp: timestamppb.Now(),
	}

	expectedError := fmt.Errorf("storage error")
	operationsMockStorage.EXPECT().
		UpdateOperationStatus(
			gomock.Any(),
			custom_profiling_operation.OperationID(operationID),
			status,
		).
		Return(expectedError).
		Times(1)

	req := &cpo_proto.UpdateOperationStatusRequest{
		ID:     operationID,
		Status: status,
	}

	resp, err := service.UpdateOperationStatus(ctx, req)

	require.Error(t, err)
	assert.Nil(t, resp)

	// Should be Internal gRPC error with wrapped storage error
	assert.Contains(t, err.Error(), "Internal")
	assert.Contains(t, err.Error(), "failed to update operation status")
	assert.Contains(t, err.Error(), "storage error")
}

func TestService_UpdateOperationStatus_ValidationErrors(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	operationsMockStorage := operation_mocks.NewMockStorage(ctrl)
	service := createTestService(t, operationsMockStorage, 2*time.Second)

	operationsMockStorage.EXPECT().UpdateOperationStatus(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	tests := []struct {
		name        string
		req         *cpo_proto.UpdateOperationStatusRequest
		expectedErr string
	}{
		{
			name:        "nil_request",
			req:         nil,
			expectedErr: "req is nil or req.ID is empty",
		},
		{
			name: "empty_operation_id",
			req: &cpo_proto.UpdateOperationStatusRequest{
				ID: "",
				Status: &cpo_proto.OperationStatus{
					State: cpo_proto.OperationState_Running,
				},
			},
			expectedErr: "req is nil or req.ID is empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := service.UpdateOperationStatus(ctx, tt.req)

			require.Error(t, err)
			assert.Nil(t, resp)

			assert.Contains(t, err.Error(), tt.expectedErr)
			assert.Contains(t, err.Error(), "InvalidArgument")
		})
	}
}

func TestService_CollectOperationsFiltersOutTerminalStates(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	operationsMockStorage := operation_mocks.NewMockStorage(ctrl)
	service := createTestService(t, operationsMockStorage, 2*time.Second)

	// Expect that ListOperations is called with a filter containing only non-terminal states
	operationsMockStorage.EXPECT().
		ListOperations(
			gomock.Any(),
			gomock.Cond(func(filter any) bool {
				opFilter, ok := filter.(*custom_profiling_operation.OperationFilter)
				if !ok {
					return false
				}

				expectedStates := custom_profiling_operation.NonTerminalStates()
				if len(opFilter.States) != len(expectedStates) {
					t.Logf("Expected %d states, got %d", len(expectedStates), len(opFilter.States))
					return false
				}

				stateMap := make(map[cpo_proto.OperationState]bool)
				for _, state := range opFilter.States {
					stateMap[state] = true
				}

				for _, expectedState := range expectedStates {
					if !stateMap[expectedState] {
						t.Logf("Expected state %v not found in filter", expectedState)
						return false
					}
				}

				return true
			}),
			gomock.Nil(),
		).
		Return(nil, nil).
		Times(1)

	operations, err := service.collectOperations(ctx)
	require.NoError(t, err)
	require.Nil(t, operations)
}
