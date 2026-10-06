package cluster_top

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Masterminds/squirrel"
	hasql "golang.yandex/hasql/sqlx"

	"github.com/yandex/perforator/perforator/pkg/lease"
	postgreslease "github.com/yandex/perforator/perforator/pkg/lease/postgres"
)

type jobQueueItem struct {
	ID          int64     `db:"id"`
	Service     string    `db:"service"`
	Generation  uint32    `db:"generation"`
	BucketCount uint16    `db:"bucket_count"`
	PodID       string    `db:"pod_id"`
	NodeID      string    `db:"node_id"`
	From        time.Time `db:"from_ts"`
	To          time.Time `db:"to_ts"`
	CreatedAt   time.Time `db:"created_at"`
}

type PgJobSelector struct {
	cluster *hasql.Cluster
	conf    JobLeaseConfig
	rows    *postgreslease.Storage
}

func NewPgJobSelector(cluster *hasql.Cluster, conf JobLeaseConfig) (*PgJobSelector, error) {
	if err := conf.Validate(); err != nil {
		return nil, err
	}

	return &PgJobSelector{cluster: cluster, conf: conf,
		rows: postgreslease.NewStorage(cluster, postgreslease.WithRowLayout(postgreslease.RowLayout{
			Table: "cluster_top_jobs", KeyColumn: "id", HolderColumn: "lease_token", ExpiresAtColumn: "lease_expires_at",
		}), postgreslease.WithExistingRowsOnly()),
	}, nil
}

func (s *PgJobSelector) SelectJob(ctx context.Context) (*SelectedJob, error) {
	claimCtx, cancel := context.WithTimeout(ctx, s.conf.OperationTimeout)
	defer cancel()
	primary, err := s.cluster.WaitForPrimary(claimCtx)
	if err != nil {
		return nil, err
	}

	var item jobQueueItem
	err = primary.DBx().GetContext(claimCtx, &item, `SELECT
        j.id, j.service, j.generation, j.pod_id, j.node_id, j.created_at,
        g.from_ts, g.to_ts, g.bucket_count
    FROM cluster_top_jobs j
    JOIN cluster_top_generations g ON g.id = j.generation
    WHERE j.status IN ('pending', 'running')
        AND (j.lease_expires_at IS NULL OR j.lease_expires_at <= clock_timestamp())
    ORDER BY j.profiles_count DESC, j.id
    LIMIT 1`)
	if err != nil {
		return nil, err
	}
	if item.BucketCount == 0 {
		return nil, fmt.Errorf("generation %d has zero bucket_count", item.Generation)
	}
	return &SelectedJob{
		Job: Job{
			ID: item.ID, Generation: int(item.Generation), BucketCount: item.BucketCount,
			Service: item.Service, PodID: item.PodID, NodeID: item.NodeID,
			TimeRange: TimeRange{From: item.From, To: item.To},
			CreatedAt: item.CreatedAt,
		},
		jobState: &pgJobState{selector: s, jobID: item.ID},
	}, nil
}

type pgJobState struct {
	selector *PgJobSelector
	jobID    int64
}

func (l *pgJobState) Lease() lease.Lease { return postgreslease.ForKey(l.selector.rows, l.jobID) }

// MarkRunning rechecks eligibility after acquisition: a previously selected candidate
// may have been completed and released by another worker in the meantime.
func (l *pgJobState) MarkRunning(ctx context.Context, token string) (time.Time, error) {
	ctx, cancel := context.WithTimeout(ctx, l.selector.conf.OperationTimeout)
	defer cancel()
	primary, err := l.selector.cluster.WaitForPrimary(ctx)
	if err != nil {
		return time.Time{}, err
	}
	var startedAt time.Time
	err = primary.DBx().GetContext(ctx, &startedAt, `UPDATE cluster_top_jobs
        SET status = 'running', started_at = clock_timestamp(), finished_at = NULL,
            execution_stats = NULL
        WHERE id = $1 AND lease_token = $2 AND lease_expires_at > clock_timestamp()
            AND status IN ('pending', 'running')
        RETURNING started_at`, l.jobID, token)
	return startedAt, err
}

func (l *pgJobState) Finalize(ctx context.Context, token, status string, stats *JobExecutionStats) error {
	if status != JobStatusDone && status != JobStatusFailed && status != JobStatusSkipped {
		return fmt.Errorf("invalid final job status %q", status)
	}
	var statsJSON []byte
	if stats != nil {
		var err error
		statsJSON, err = json.Marshal(stats)
		if err != nil {
			return fmt.Errorf("failed to marshal execution stats: %w", err)
		}
	}
	ctx, cancel := context.WithTimeout(ctx, l.selector.conf.OperationTimeout)
	defer cancel()
	primary, err := l.selector.cluster.WaitForPrimary(ctx)
	if err != nil {
		return err
	}
	query, args, err := squirrel.Update("cluster_top_jobs").
		Set("status", status).
		Set("finished_at", squirrel.Expr("clock_timestamp()")).
		Set("execution_stats", statsJSON).
		Where(squirrel.Eq{"id": l.jobID, "lease_token": token}).
		Where(squirrel.Expr("lease_expires_at > clock_timestamp()")).
		Where(squirrel.Eq{"status": JobStatusRunning}).
		PlaceholderFormat(squirrel.Dollar).ToSql()
	if err != nil {
		return err
	}
	result, err := primary.DBx().ExecContext(ctx, query, args...)
	return checkLeaseUpdate(result, err)
}

func checkLeaseUpdate(result sql.Result, err error) error {
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count != 1 {
		return ErrJobLeaseLost
	}
	return nil
}
