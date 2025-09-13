package custom_profiling_operation

import (
	"sync"
	"time"

	"github.com/yandex/perforator/perforator/pkg/pubsub"
	cpo_proto "github.com/yandex/perforator/perforator/proto/custom_profiling_operation"
)

type operationsSnapshot struct {
	podToOperations  map[string][]*cpo_proto.Operation
	nodeToOperations map[string][]*cpo_proto.Operation
}

func buildCustomProfilingOperationSnapshot(batch []*cpo_proto.Operation) *operationsSnapshot {
	res := operationsSnapshot{
		podToOperations:  make(map[string][]*cpo_proto.Operation),
		nodeToOperations: make(map[string][]*cpo_proto.Operation),
	}

	appendOperation := func(key string, operation *cpo_proto.Operation, operations map[string][]*cpo_proto.Operation) {
		batch, ok := operations[key]
		if !ok {
			batch = make([]*cpo_proto.Operation, 0, 1)
		}

		batch = append(batch, operation)
		operations[key] = batch
	}

	appendPodOperation := func(pod string, operation *cpo_proto.Operation) {
		appendOperation(pod, operation, res.podToOperations)
	}

	appendNodeOperation := func(node string, operation *cpo_proto.Operation) {
		appendOperation(node, operation, res.nodeToOperations)
	}

	for _, operation := range batch {
		if operation == nil || operation.Spec == nil || operation.Spec.Target == nil || operation.Spec.Target.Target == nil {
			continue
		}

		switch target := operation.Spec.Target.Target.(type) {
		case *cpo_proto.Target_NodeProcess:
			if target.NodeProcess != nil && target.NodeProcess.Host != "" {
				appendNodeOperation(target.NodeProcess.Host, operation)
			}
		case *cpo_proto.Target_NodeCgroup:
			if target.NodeCgroup != nil && target.NodeCgroup.Host != "" {
				appendNodeOperation(target.NodeCgroup.Host, operation)
			}
		case *cpo_proto.Target_Pod:
			if target.Pod != nil && target.Pod.Pod != "" {
				appendPodOperation(target.Pod.Pod, operation)
			}
		}
	}

	return &res
}

type operationsSnapshotManager struct {
	mutex     sync.RWMutex
	snapshot  *operationsSnapshot
	timestamp time.Time
	pubSub    *pubsub.PubSub[*operationsSnapshot]
}

func newOperationsSnapshotManager() *operationsSnapshotManager {
	return &operationsSnapshotManager{
		pubSub: pubsub.NewPubSub[*operationsSnapshot](),
	}
}

func (m *operationsSnapshotManager) subscribeToUpdates() *pubsub.Subscription[*operationsSnapshot] {
	// Lets use some capacity to avoid blocking new subscriptions
	return m.pubSub.Subscribe(pubsub.WithChanCapacity(5))
}

func (m *operationsSnapshotManager) updateSnapshot(snapshot *operationsSnapshot) {
	now := time.Now()
	m.mutex.Lock()
	m.snapshot = snapshot
	m.timestamp = now
	m.mutex.Unlock() // unlock here to avoid blocking getSnapshot because of Publish(snapshot)

	m.pubSub.Publish(snapshot)
}

func (m *operationsSnapshotManager) getSnapshot() *operationsSnapshot {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.snapshot
}

func (m *operationsSnapshotManager) getSnapshotTimestamp() time.Time {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.timestamp
}

func obtainOperationsFromSnapshot(snapshot *operationsSnapshot, filter *cpo_proto.PollOperationsFilter) []*cpo_proto.Operation {
	deduplicatedOperations := make(map[string]*cpo_proto.Operation)

	for _, operation := range snapshot.nodeToOperations[filter.Host] {
		deduplicatedOperations[operation.ID] = operation
	}

	for _, pod := range filter.Pods {
		for _, operation := range snapshot.podToOperations[pod] {
			deduplicatedOperations[operation.ID] = operation
		}
	}

	res := make([]*cpo_proto.Operation, 0, len(deduplicatedOperations))
	for _, operation := range deduplicatedOperations {
		res = append(res, operation)
	}

	return res
}
