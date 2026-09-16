package gc

import (
	"context"
	"fmt"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	hasql "golang.yandex/hasql/sqlx"
)

const (
	mainTable       = "cluster_top_v3"
	byFunctionTable = "cluster_top_by_function_v3"
)

type Generation struct {
	ID             uint32    `db:"id"`
	To             time.Time `db:"to_ts"`
	Status         string    `db:"status"`
	BucketCount    uint32    `db:"bucket_count"`
	HasPendingJobs bool      `db:"has_pending_jobs"`
}

type storage struct {
	postgres   *hasql.Cluster
	clickhouse driver.Conn
	timeout    time.Duration
}

func NewStorage(pg *hasql.Cluster, ch driver.Conn, timeout time.Duration) *storage {
	return &storage{postgres: pg, clickhouse: ch, timeout: timeout}
}

func (s *storage) ListGenerations(ctx context.Context) ([]Generation, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	primary, err := s.postgres.WaitForPrimary(ctx)
	if err != nil {
		return nil, err
	}
	var result []Generation
	err = primary.DBx().SelectContext(ctx, &result, `
 SELECT g.id, g.to_ts, COALESCE(g.status, 'finished') AS status,
        COALESCE(g.bucket_count, 0) AS bucket_count,
        EXISTS (SELECT 1 FROM cluster_top_jobs j WHERE j.generation = g.id AND j.status = 'pending') AS has_pending_jobs
 FROM cluster_top_generations g ORDER BY g.to_ts, g.id`)
	return result, err
}

func (s *storage) MarkDeleting(ctx context.Context, generation uint32, cutoff time.Time) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	primary, err := s.postgres.WaitForPrimary(ctx)
	if err != nil {
		return false, err
	}
	// Recheck eligibility in the transition statement, including the scheduler's
	// ID watermark. Already-deleting generations are resumed regardless of TTL.
	result, err := primary.DBx().ExecContext(ctx, `
 UPDATE cluster_top_generations g SET status = 'deleting'
 WHERE g.id = $1 AND g.bucket_count > 0
   AND NOT EXISTS (SELECT 1 FROM cluster_top_jobs j WHERE j.generation = g.id AND j.status = 'pending')
   AND (g.status = 'deleting' OR (
     g.status = 'finished' AND g.to_ts < $2
     AND g.id < (SELECT MAX(id) FROM cluster_top_generations WHERE COALESCE(status, 'finished') = 'finished')
   ))`, generation, cutoff)
	if err != nil {
		return false, err
	}
	count, err := result.RowsAffected()
	return count != 0, err
}

// DeleteData waits for each replicated mutation on the accepting replica before
// PostgreSQL metadata can be removed. Other replicas apply the persisted mutation
// independently; a failed or uncertain response can be retried for this generation.
func (s *storage) DeleteData(ctx context.Context, generation uint32) error {
	for _, table := range []string{mainTable, byFunctionTable} {
		if err := context.Cause(ctx); err != nil {
			return err
		}
		operationCtx, cancel := context.WithTimeout(ctx, s.timeout)
		err := s.clickhouse.Exec(operationCtx,
			"ALTER TABLE "+table+" DELETE WHERE generation = ? SETTINGS mutations_sync = 1", generation)
		cancel()
		if err != nil {
			return fmt.Errorf("delete generation %d from %s: %w", generation, table, err)
		}
	}
	return nil
}

func (s *storage) DeleteJobs(ctx context.Context, generation uint32, batchSize uint32) (int64, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	primary, err := s.postgres.WaitForPrimary(ctx)
	if err != nil {
		return 0, err
	}
	result, err := primary.DBx().ExecContext(ctx, `
 WITH batch AS (
     SELECT j.id
     FROM cluster_top_jobs j
     JOIN cluster_top_generations g ON g.id = j.generation
     WHERE j.generation = $1 AND g.status = 'deleting'
     LIMIT $2
 )
 DELETE FROM cluster_top_jobs j
 USING batch b
 WHERE j.id = b.id AND j.generation = $1`, generation, batchSize)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (s *storage) DeleteGeneration(ctx context.Context, generation uint32) error {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	primary, err := s.postgres.WaitForPrimary(ctx)
	if err != nil {
		return err
	}
	result, err := primary.DBx().ExecContext(ctx, `
 DELETE FROM cluster_top_generations g
 WHERE g.id = $1 AND g.status = 'deleting' AND g.bucket_count > 0
   AND NOT EXISTS (SELECT 1 FROM cluster_top_jobs WHERE generation = g.id)`, generation)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count != 1 {
		return fmt.Errorf("generation %d is not ready for metadata deletion", generation)
	}
	return nil
}
