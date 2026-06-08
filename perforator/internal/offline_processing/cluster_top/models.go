package cluster_top

import (
	"context"
	"time"

	"github.com/yandex/perforator/perforator/pkg/storage/cluster_top/aggregated"
)

type TimeRange struct {
	From time.Time
	To   time.Time
}

func workloadKey(podID, nodeID string) string {
	if podID != "" {
		return podID
	}
	return nodeID
}

type Job struct {
	ID         int64
	Generation int
	Service    string
	PodID      string
	NodeID     string
	TimeRange  TimeRange
}

func (j Job) WorkloadKey() string {
	return workloadKey(j.PodID, j.NodeID)
}

type SelectedJob struct {
	Job Job

	finalize func(ctx context.Context, processingErr error)
}

func (s *SelectedJob) Finalize(ctx context.Context, processingErr error) {
	if s.finalize != nil {
		s.finalize(ctx, processingErr)
	}
}

type JobSelector interface {
	SelectJob(ctx context.Context) (*SelectedJob, error)
}

type Function = aggregated.Function

type ServicePerfTop = aggregated.ServicePerfTop

type ClusterPerfTopAggregator interface {
	Save(ctx context.Context, servicePerfTop *ServicePerfTop) error

	Print(ctx context.Context) error
}
