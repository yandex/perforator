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
	"github.com/yandex/perforator/perforator/pkg/storage/bundle"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

const clusterTopSchedulerLeaseName = "cluster_top_scheduler"

const clusterTopSystemName = "perforator"

const maxJobsInsertBatchSize = 10000

type generationStatus string

const (
	generationStatusScheduled generationStatus = "scheduled"
	generationStatusFinished  generationStatus = "finished"
)

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
}

func (c *Config) FillDefault() {
	if c.LeaseTTL == 0 {
		c.LeaseTTL = 30 * time.Second
	}
	if c.MaxConflictErrors == 0 {
		c.MaxConflictErrors = 3
	}
}

type Scheduler struct {
	l       xlog.Logger
	reg     metrics.Registry
	storage *bundle.StorageBundle
	conf    *Config

	schedulerSuccess          metrics.Counter
	schedulerErrors           metrics.Counter
	finisherSuccess           metrics.Counter
	finisherErrors            metrics.Counter
	schedulingThrottled       metrics.Counter
	pendingJobsGauge          metrics.IntGauge
	scheduledGenerationsGauge metrics.IntGauge
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
	conf *Config,
) *Scheduler {
	r := reg.WithPrefix("cluster_top_scheduler")

	return &Scheduler{
		l:                         l.WithName("Scheduler"),
		reg:                       r,
		storage:                   storage,
		conf:                      conf,
		schedulerSuccess:          r.WithTags(map[string]string{"component": "generation_scheduler", "status": "success"}).Counter("iterations.count"),
		schedulerErrors:           r.WithTags(map[string]string{"component": "generation_scheduler", "status": "error"}).Counter("iterations.count"),
		finisherSuccess:           r.WithTags(map[string]string{"component": "generation_finisher", "status": "success"}).Counter("iterations.count"),
		finisherErrors:            r.WithTags(map[string]string{"component": "generation_finisher", "status": "error"}).Counter("iterations.count"),
		schedulingThrottled:       r.Counter("scheduling.throttled.count"),
		pendingJobsGauge:          r.IntGauge("jobs.pending.count"),
		scheduledGenerationsGauge: r.IntGauge("generations.scheduled.count"),
	}
}

func (s *Scheduler) hasScheduledGeneration(ctx context.Context) (bool, error) {
	primary, err := s.storage.DBs.PostgresCluster.WaitForPrimary(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to wait for primary postgres: %w", err)
	}

	return hasScheduledGeneration(ctx, primary.DBx())
}

func hasScheduledGeneration(ctx context.Context, q sqlGetter) (bool, error) {
	var exists bool
	err := q.GetContext(
		ctx,
		&exists,
		`SELECT EXISTS(SELECT 1 FROM cluster_top_generations WHERE status = $1)`,
		generationStatusScheduled,
	)
	if err != nil {
		return false, fmt.Errorf("failed to check scheduled generations: %w", err)
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

	hasScheduled, err := hasScheduledGeneration(ctx, tx)
	if err != nil {
		return 0, err
	}
	if hasScheduled {
		return 0, errUnfinishedGenerationExists
	}

	err = tx.QueryRowContext(ctx,
		`INSERT INTO cluster_top_generations (id, from_ts, to_ts, status) 
		 VALUES ($1, $2, $3, $4) 
		 ON CONFLICT (id) DO NOTHING 
		 RETURNING id`,
		generationID, start, end, generationStatusScheduled,
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
	hasScheduled, err := s.hasScheduledGeneration(ctx)
	if err != nil {
		return fmt.Errorf("failed to check unfinished generations: %w", err)
	}
	if hasScheduled {
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
		log.Time("start", targetStart),
		log.Time("end", targetEnd),
		log.Int("services_count", countUniqueServices(jobs)),
		log.Int("jobs_count", jobCount),
	)

	return nil
}

func (s *Scheduler) finishGenerations(ctx context.Context) error {
	primary, err := s.storage.DBs.PostgresCluster.WaitForPrimary(ctx)
	if err != nil {
		return fmt.Errorf("failed to wait for primary postgres: %w", err)
	}

	tx, err := primary.DBx().BeginTxx(ctx, repeatableReadTxOptions)
	if err != nil {
		return fmt.Errorf("failed to start postgres tx: %w", err)
	}
	defer tx.Rollback()

	var scheduledIDs []int
	query, args, err := sq.Select("id").
		From("cluster_top_generations").
		Where(sq.Eq{"status": generationStatusScheduled}).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return fmt.Errorf("failed to build in_progress query: %w", err)
	}

	err = tx.SelectContext(ctx, &scheduledIDs, query, args...)
	if err != nil {
		return fmt.Errorf("failed to fetch in_progress generations: %w", err)
	}

	errs := []error{}
	totalPendingCount := 0
	remainingScheduledCount := len(scheduledIDs)
	for _, id := range scheduledIDs {
		var pendingCount int
		err = tx.GetContext(ctx, &pendingCount,
			`SELECT count(*) FROM cluster_top_jobs WHERE generation = $1 AND status = 'pending'`,
			id,
		)
		if err != nil {
			s.l.Error(ctx, "Failed to count pending jobs for in_progress generation", log.Int("id", id), log.Error(err))
			errs = append(errs, err)
			continue
		}

		totalPendingCount += pendingCount

		if pendingCount == 0 {
			_, err = tx.ExecContext(ctx,
				`UPDATE cluster_top_generations SET status = $1 WHERE id = $2`,
				generationStatusFinished, id,
			)
			if err != nil {
				s.l.Error(ctx, "Failed to update generation status to finished", log.Int("id", id), log.Error(err))
				errs = append(errs, err)
			} else {
				remainingScheduledCount--
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit finish generations tx: %w", err)
	}

	s.pendingJobsGauge.Set(int64(totalPendingCount))
	s.scheduledGenerationsGauge.Set(int64(remainingScheduledCount))

	return errors.Join(errs...)
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
			return ctx.Err()
		case <-ticker.C:
			if err := iteration(); err != nil {
				return err
			}
		}
	}
}

func (s *Scheduler) runGenerationFinisher(ctx context.Context) error {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	iteration := func() {
		if err := s.finishGenerations(ctx); err != nil {
			s.l.Error(ctx, "Failed to run generation finisher tick", log.Error(err))
			s.finisherErrors.Inc()
		} else {
			s.finisherSuccess.Inc()
		}
	}

	iteration()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			iteration()
		}
	}
}

func (s *Scheduler) Run(ctx context.Context) error {
	holderID, err := lease.BuildPerProcessHolderID()
	if err != nil {
		return fmt.Errorf("failed to build lease holder ID: %w", err)
	}

	return lease.LockAndRun(
		ctx,
		s.l,
		s.storage.LeaseStorage,
		clusterTopSchedulerLeaseName,
		holderID,
		func(leaseCtx context.Context) {
			g, gCtx := errgroup.WithContext(leaseCtx)

			g.Go(func() error {
				return s.runScheduler(gCtx)
			})

			g.Go(func() error {
				return s.runGenerationFinisher(gCtx)
			})

			if err := g.Wait(); err != nil {
				s.l.Error(leaseCtx, "Scheduler stopped", log.Error(err))
			}
		},
		lease.WithTTL(s.conf.LeaseTTL),
	)
}
