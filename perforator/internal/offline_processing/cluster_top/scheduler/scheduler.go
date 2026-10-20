package scheduler

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"golang.org/x/sync/errgroup"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/library/go/core/metrics"
	"github.com/yandex/perforator/perforator/internal/xmetrics"
	"github.com/yandex/perforator/perforator/pkg/lease"
	postgreslease "github.com/yandex/perforator/perforator/pkg/lease/postgres"
	"github.com/yandex/perforator/perforator/pkg/storage/bundle"
	"github.com/yandex/perforator/perforator/pkg/storage/cluster_top/generations"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

const clusterTopSchedulerLeaseName = "cluster_top_scheduler"

const clusterTopSystemName = "perforator"

const maxJobsInsertBatchSize = 10000

const DefaultBucketCount uint16 = 16

type GenerationID = uint32

var errGenerationAlreadyExists = errors.New("generation already exists")

var errUnfinishedGenerationExists = errors.New("unfinished generation exists")

var repeatableReadTxOptions = &sql.TxOptions{
	Isolation: sql.LevelRepeatableRead,
}

type Config struct {
	GenerationInterval time.Duration
	ProfileLag         time.Duration
	LeaseTTL           time.Duration
	MaxConflictErrors  uint32
	BucketCount        uint16
	ServingTables      ServingTablesConfig
}

func (c *Config) FillDefault() {
	c.ServingTables.FillDefault()
	if c.BucketCount == 0 {
		c.BucketCount = DefaultBucketCount
	}
	if c.LeaseTTL == 0 {
		c.LeaseTTL = 30 * time.Second
	}
	if c.MaxConflictErrors == 0 {
		c.MaxConflictErrors = 3
	}
}

type Scheduler struct {
	l                    xlog.Logger
	storage              *bundle.StorageBundle
	conf                 *Config
	generationController *GenerationController

	schedulerSuccess    metrics.Counter
	schedulerErrors     metrics.Counter
	schedulingThrottled metrics.Counter
}

type jobInsertRow struct {
	generation    int32
	service       string
	podID         string
	nodeID        string
	profilesCount uint64
}

func NewScheduler(
	l xlog.Logger,
	reg xmetrics.Registry,
	storage *bundle.StorageBundle,
	snapshots snapshotBuilder,
	conf *Config,
) *Scheduler {
	conf.FillDefault()
	r := reg.WithPrefix("cluster_top_scheduler")

	return &Scheduler{
		l:                    l.WithName("Scheduler"),
		storage:              storage,
		conf:                 conf,
		generationController: newGenerationController(l, r, storage.DBs.PostgresCluster, snapshots, conf.ServingTables),
		schedulerSuccess:     r.WithTags(map[string]string{"component": "generation_scheduler", "status": "success"}).Counter("iterations.count"),
		schedulerErrors:      r.WithTags(map[string]string{"component": "generation_scheduler", "status": "error"}).Counter("iterations.count"),
		schedulingThrottled:  r.Counter("scheduling.throttled.count"),
	}
}

func (s *Scheduler) hasUnfinishedGeneration(ctx context.Context) (bool, error) {
	primary, err := s.storage.DBs.PostgresCluster.WaitForPrimary(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to wait for primary postgres: %w", err)
	}

	return hasUnfinishedGeneration(ctx, primary.DBx())
}

func hasUnfinishedGeneration(ctx context.Context, q sqlGetter) (bool, error) {
	var exists bool
	err := q.GetContext(
		ctx,
		&exists,
		`SELECT EXISTS(SELECT 1 FROM cluster_top_generations WHERE status IN ($1, $2))`,
		generations.StatusScheduled,
		generations.StatusFinalizing,
	)
	if err != nil {
		return false, fmt.Errorf("failed to check unfinished generations: %w", err)
	}

	return exists, nil
}

type sqlGetter interface {
	GetContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error
}

func (s *Scheduler) getLastGeneration(ctx context.Context) (lastID int32, maxTo time.Time, err error) {
	primary, err := s.storage.DBs.PostgresCluster.WaitForPrimary(ctx)
	if err != nil {
		return 0, time.Time{}, fmt.Errorf("failed to wait for primary postgres: %w", err)
	}

	err = primary.DBx().QueryRowContext(
		ctx,
		`SELECT 
			COALESCE((SELECT MAX(id) FROM cluster_top_generations), 0), 
			COALESCE((SELECT MAX(to_ts) FROM cluster_top_generations), 'epoch'::timestamptz)`,
	).Scan(&lastID, &maxTo)

	if err != nil {
		return 0, time.Time{}, fmt.Errorf("failed to execute query: %w", err)
	}

	return lastID, maxTo, nil
}

func discoveredJobsToInsertRows(
	generationID int32,
	jobs []discoveredJob,
) []jobInsertRow {
	rows := make([]jobInsertRow, 0, len(jobs))
	for _, job := range jobs {
		rows = append(rows, jobInsertRow{
			generation:    generationID,
			service:       job.service,
			podID:         job.podID,
			nodeID:        job.nodeID,
			profilesCount: job.profilesCount,
		})
	}
	return rows
}

func (s *Scheduler) createGeneration(ctx context.Context, generationID int32, start, end time.Time, jobs []discoveredJob) (int, error) {
	insertRows := discoveredJobsToInsertRows(generationID, jobs)

	primary, err := s.storage.DBs.PostgresCluster.WaitForPrimary(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to wait for primary postgres: %w", err)
	}

	tx, err := primary.DBx().BeginTxx(ctx, repeatableReadTxOptions)
	if err != nil {
		return 0, fmt.Errorf("failed to start postgres tx: %w", err)
	}
	defer tx.Rollback()

	hasUnfinished, err := hasUnfinishedGeneration(ctx, tx)
	if err != nil {
		return 0, err
	}
	if hasUnfinished {
		return 0, errUnfinishedGenerationExists
	}

	err = tx.QueryRowContext(ctx,
		`INSERT INTO cluster_top_generations (id, from_ts, to_ts, status, bucket_count) 
		 VALUES ($1, $2, $3, $4, $5) 
		 ON CONFLICT (id) DO NOTHING 
		 RETURNING id`,
		generationID, start, end, generations.StatusScheduled, s.conf.BucketCount,
	).Scan(&generationID)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, errGenerationAlreadyExists
		}
		return 0, fmt.Errorf("failed to insert generation: %w", err)
	}

	for i := 0; i < len(insertRows); i += maxJobsInsertBatchSize {
		end := min(i+maxJobsInsertBatchSize, len(insertRows))
		batch := insertRows[i:end]

		builder := sq.StatementBuilder.PlaceholderFormat(sq.Dollar).
			Insert("cluster_top_jobs").
			Columns(
				"generation",
				"service",
				"pod_id",
				"node_id",
				"profiles_count",
				"status",
			)

		for _, job := range batch {
			builder = builder.Values(
				job.generation,
				job.service,
				job.podID,
				job.nodeID,
				job.profilesCount,
				"pending",
			)
		}

		query, args, err := builder.ToSql()
		if err != nil {
			return 0, fmt.Errorf("failed to build postgres insert query: %w", err)
		}

		_, err = tx.ExecContext(ctx, query, args...)
		if err != nil {
			return 0, fmt.Errorf("failed to insert jobs: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}

	return len(insertRows), nil
}

func (s *Scheduler) tryScheduleGeneration(ctx context.Context) error {
	now := time.Now()

	lastID, maxTo, err := s.getLastGeneration(ctx)
	if err != nil {
		return fmt.Errorf("failed to get last generation: %w", err)
	}

	latestAllowedEnd := now.Add(-s.conf.ProfileLag).Truncate(s.conf.GenerationInterval)

	var targetStart, targetEnd time.Time
	if maxTo.Unix() <= 0 { // 'epoch' fallback or empty
		targetEnd = latestAllowedEnd
		targetStart = targetEnd.Add(-s.conf.GenerationInterval)
	} else {
		targetStart = maxTo
		targetEnd = targetStart.Add(s.conf.GenerationInterval)

		if targetEnd.Before(latestAllowedEnd.Add(-s.conf.GenerationInterval)) {
			s.l.Warn(ctx, "Skipping stale generations",
				log.Time("old_target_end", targetEnd),
				log.Time("new_target_end", latestAllowedEnd),
				log.Time("max_to", maxTo),
			)
			targetEnd = latestAllowedEnd
			targetStart = targetEnd.Add(-s.conf.GenerationInterval)
		}
	}

	if now.Before(targetEnd.Add(s.conf.ProfileLag)) {
		s.l.Info(ctx, "It's too early to schedule next generation",
			log.Time("target_start", targetStart),
			log.Time("target_end", targetEnd),
			log.Time("now", now),
		)
		return nil
	}

	// Fast-path backpressure: skip before discoverJobs (ClickHouse). schedulingThrottled
	// is incremented only here — intentional wait for the current generation to finish.
	// createGeneration re-checks under RepeatableRead on primary.
	hasUnfinished, err := s.hasUnfinishedGeneration(ctx)
	if err != nil {
		return fmt.Errorf("failed to check unfinished generations: %w", err)
	}
	if hasUnfinished {
		s.l.Info(ctx, "Skipping generation scheduling: unfinished generation exists")
		s.schedulingThrottled.Inc()
		return nil
	}

	generationID := lastID + 1

	s.l.Info(ctx, "Trying to schedule new generation",
		log.Time("start", targetStart),
		log.Time("end", targetEnd),
		log.Int("generation_id", int(generationID)),
	)

	jobs, err := s.discoverJobs(ctx, targetStart, targetEnd)
	if err != nil {
		return fmt.Errorf("failed to discover jobs: %w", err)
	}

	jobCount, err := s.createGeneration(ctx, generationID, targetStart, targetEnd, jobs)
	if err != nil {
		if errors.Is(err, errUnfinishedGenerationExists) {
			s.l.Info(ctx, "Skipping generation scheduling: unfinished generation appeared before insert")
			return nil
		}
		return fmt.Errorf("failed to create generation: %w", err)
	}

	s.l.Info(ctx, "Successfully created new generation",
		log.UInt32("bucket_count", uint32(s.conf.BucketCount)),
		log.Time("start", targetStart),
		log.Time("end", targetEnd),
		log.Int("services_count", countUniqueServices(jobs)),
		log.Int("jobs_count", jobCount),
	)

	return nil
}

func (s *Scheduler) runScheduler(ctx context.Context) error {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	conflictErrorsCount := 0

	iteration := func() error {
		if err := s.tryScheduleGeneration(ctx); err != nil {
			s.schedulerErrors.Inc()

			if errors.Is(err, errGenerationAlreadyExists) {
				conflictErrorsCount++
				s.l.Warn(ctx, "Generation already exists, skipping creation", log.Error(err), log.Int("consecutive_conflicts", conflictErrorsCount))

				if uint32(conflictErrorsCount) >= s.conf.MaxConflictErrors {
					s.l.Error(ctx, "Too many consecutive generation conflicts, shutting down to prevent split-brain")
					return fmt.Errorf("exceeded max consecutive generation conflicts (%d)", s.conf.MaxConflictErrors)
				}
			} else {
				conflictErrorsCount = 0
				s.l.Error(ctx, "Failed to run scheduler tick", log.Error(err))
			}
		} else {
			conflictErrorsCount = 0
			s.schedulerSuccess.Inc()
		}

		return nil
	}

	if err := iteration(); err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return context.Cause(ctx)
		case <-ticker.C:
			if err := iteration(); err != nil {
				return err
			}
		}
	}
}

func (s *Scheduler) Run(ctx context.Context) error {
	if err := s.conf.ServingTables.Validate(); err != nil {
		return err
	}
	var runErr error
	err := lease.LockAndRun(
		ctx,
		s.l,
		postgreslease.ForKey(s.storage.LeaseStorage, clusterTopSchedulerLeaseName),
		func(leaseCtx context.Context, _ string) {
			g, gCtx := errgroup.WithContext(leaseCtx)

			g.Go(func() error {
				return s.runScheduler(gCtx)
			})

			g.Go(func() error {
				return s.generationController.Run(gCtx)
			})

			runErr = g.Wait()
			if runErr != nil {
				s.l.Error(leaseCtx, "Scheduler stopped", log.Error(runErr))
				if cause := context.Cause(leaseCtx); cause != nil {
					runErr = cause
				}
			}
		},
		lease.WithTTL(s.conf.LeaseTTL),
	)
	if err != nil {
		return err
	}
	return runErr
}
