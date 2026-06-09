package cluster_top

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	hasql "golang.yandex/hasql/sqlx"
)

type jobQueueItem struct {
	ID         int64     `db:"id"`
	Service    string    `db:"service"`
	Generation uint32    `db:"generation"`
	PodID      string    `db:"pod_id"`
	NodeID     string    `db:"node_id"`
	From       time.Time `db:"from_ts"`
	To         time.Time `db:"to_ts"`
	CreatedAt  time.Time `db:"created_at"`
}

type PgJobSelector struct {
	cluster *hasql.Cluster
}

func NewPgJobSelector(cluster *hasql.Cluster) *PgJobSelector {
	return &PgJobSelector{
		cluster: cluster,
	}
}

func (s *PgJobSelector) SelectJob(ctx context.Context) (*SelectedJob, error) {
	primary, err := s.cluster.WaitForPrimary(ctx)
	if err != nil {
		return nil, err
	}

	tx, err := primary.DBx().BeginTxx(ctx, &sql.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to start tx: %w", err)
	}

	var queueItem jobQueueItem
	err = tx.GetContext(
		ctx,
		&queueItem,
		`SELECT
			j.id,
			j.service,
			j.generation,
			j.pod_id,
			j.node_id,
			j.created_at,
			g.from_ts,
			g.to_ts
		FROM cluster_top_jobs AS j
		INNER JOIN cluster_top_generations AS g ON g.id = j.generation
		WHERE
			j.status = 'pending'
		ORDER BY
			j.profiles_count DESC
		LIMIT 1
		FOR UPDATE OF j SKIP LOCKED
		`,
	)
	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}

	var startedAt time.Time
	err = tx.GetContext(
		ctx,
		&startedAt,
		`UPDATE cluster_top_jobs
		SET started_at = clock_timestamp()
		WHERE id = $1
		RETURNING started_at`,
		queueItem.ID,
	)
	if err != nil {
		_ = tx.Rollback()
		return nil, fmt.Errorf("failed to set started_at: %w", err)
	}

	job := Job{
		ID:         queueItem.ID,
		Generation: int(queueItem.Generation),
		Service:    queueItem.Service,
		PodID:      queueItem.PodID,
		NodeID:     queueItem.NodeID,
		TimeRange: TimeRange{
			From: queueItem.From,
			To:   queueItem.To,
		},
		CreatedAt: queueItem.CreatedAt,
		StartedAt: startedAt,
	}

	return &SelectedJob{
		Job: job,
		finalize: func(ctx context.Context, processingErr error, stats *JobExecutionStats) {
			newStatus := "done"
			if processingErr != nil {
				newStatus = "failed"
			}

			var executionStatsJSON []byte
			if stats != nil {
				var marshalErr error
				executionStatsJSON, marshalErr = json.Marshal(stats)
				if marshalErr != nil {
					_ = tx.Rollback()
					return
				}
			}

			_, finalizationErr := tx.ExecContext(
				ctx,
				`UPDATE cluster_top_jobs
				SET
					status = $2,
					finished_at = clock_timestamp(),
					execution_stats = $3
				WHERE
					id = $1`,
				job.ID,
				newStatus,
				executionStatsJSON,
			)
			if finalizationErr == nil {
				_ = tx.Commit()
			} else {
				_ = tx.Rollback()
			}
		},
	}, nil
}
