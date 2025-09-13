package custom_profiling_operation

import (
	"context"
	"hash/fnv"

	"github.com/yandex/perforator/perforator/pkg/pubsub"
	cpo_proto "github.com/yandex/perforator/perforator/proto/custom_profiling_operation"
)

func hashOperationsSet(operations []*cpo_proto.Operation) uint64 {
	if len(operations) == 0 {
		return 0
	}

	var result uint64
	hasher := fnv.New64a()

	for _, operation := range operations {
		hasher.Reset()
		// hasher.Write never returns an error
		_, _ = hasher.Write([]byte(operation.ID))

		result ^= hasher.Sum64()
	}

	return result
}

// longPollRequestSnapshotWatcher provides a long polling mechanism for tracking changes in operation snapshots.
// It allows clients to subscribe to changes and receive updates only when there are actual changes in the operations set
type longPollRequestSnapshotWatcher struct {
	subscription    *pubsub.Subscription[*operationsSnapshot]
	snapshotManager *operationsSnapshotManager
}

func newLongPollRequestSnapshotWatcher(snapshotManager *operationsSnapshotManager) *longPollRequestSnapshotWatcher {
	return &longPollRequestSnapshotWatcher{
		snapshotManager: snapshotManager,
	}
}

func buildPollOperationsResponse(operations []*cpo_proto.Operation) *cpo_proto.PollOperationsResponse {
	return &cpo_proto.PollOperationsResponse{
		Operations: operations,
		NextLongPollingData: &cpo_proto.LongPollingData{
			Data: &cpo_proto.LongPollingData_OperationsVersion{
				OperationsVersion: hashOperationsSet(operations),
			},
		},
	}
}

func (w *longPollRequestSnapshotWatcher) longPollOperations(ctx context.Context, req *cpo_proto.PollOperationsRequest) *cpo_proto.PollOperationsResponse {
	subscription := w.snapshotManager.subscribeToUpdates()
	defer subscription.Close()

	// fast path
	candidateResponse := buildPollOperationsResponse(obtainOperationsFromSnapshot(w.snapshotManager.getSnapshot(), req.Filter))
	if candidateResponse.NextLongPollingData.GetOperationsVersion() != req.LongPollingData.GetOperationsVersion() {
		return candidateResponse
	}

	for {
		select {
		case <-ctx.Done():
			return buildPollOperationsResponse(obtainOperationsFromSnapshot(w.snapshotManager.getSnapshot(), req.Filter))

		case snapshot := <-subscription.Chan():
			currentResponse := buildPollOperationsResponse(obtainOperationsFromSnapshot(snapshot, req.Filter))
			if currentResponse.NextLongPollingData.GetOperationsVersion() != req.LongPollingData.GetOperationsVersion() {
				return currentResponse
			}
		}
	}
}
