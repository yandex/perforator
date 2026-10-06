package cluster_top

import (
	"context"
	"time"

	"github.com/yandex/perforator/perforator/pkg/lease"
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
	ID          int64
	Generation  int
	BucketCount uint16
	Service     string
	PodID       string
	NodeID      string
	TimeRange   TimeRange
	CreatedAt   time.Time
	StartedAt   time.Time
}

func (j Job) WorkloadKey() string {
	return workloadKey(j.PodID, j.NodeID)
}

var ErrJobLeaseLost = lease.ErrLeaseLost

// SelectedJob is a candidate, not an acquired job. Its eligibility must be
// checked by MarkRunning after LockAndRun has acquired its lease.
type SelectedJob struct {
	Job Job
	jobState
}

type jobState interface {
	Lease() lease.Lease
	MarkRunning(context.Context, string) (time.Time, error)
	Finalize(context.Context, string, string, *JobExecutionStats) error
}

type JobSelector interface {
	SelectJob(ctx context.Context) (*SelectedJob, error)
}

type Function = aggregated.Function

type JobResult = aggregated.JobResult

type ClusterPerfTopAggregator interface {
	Save(ctx context.Context, result *JobResult) error

	Print(ctx context.Context) error
}
