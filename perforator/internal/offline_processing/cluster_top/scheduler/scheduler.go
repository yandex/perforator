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

const maxServicesInsertBatchSize = 10000

type generationStatus string

const (
	generationStatusScheduled generationStatus = "scheduled"
	generationStatusFinished  generationStatus = "finished"
)

var errGenerationAlreadyExists = errors.New("generation already exists")

type Config struct {
	GenerationInterval time.Duration
	ProfileLag         time.Duration
	MaxServices        int
	HeavyPercent       float64
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

	schedulerSuccess metrics.Counter
	schedulerErrors  metrics.Counter
	finisherSuccess  metrics.Counter
	finisherErrors   metrics.Counter
}

type serviceInfo struct {
	name         string
	profileCount uint64
	heavy        bool
}

func NewScheduler(
	l xlog.Logger,
	reg xmetrics.Registry,
	storage *bundle.StorageBundle,
	conf *Config,
) *Scheduler {
	r := reg.WithPrefix("cluster_top_scheduler")

	return &Scheduler{
		l:                l.WithName("Scheduler"),
		reg:              r,
		storage:          storage,
		conf:             conf,
		schedulerSuccess: r.WithTags(map[string]string{"component": "generation_scheduler", "status": "success"}).Counter("iterations.count"),
		schedulerErrors:  r.WithTags(map[string]string{"component": "generation_scheduler", "status": "error"}).Counter("iterations.count"),
		finisherSuccess:  r.WithTags(map[string]string{"component": "generation_finisher", "status": "success"}).Counter("iterations.count"),
		finisherErrors:   r.WithTags(map[string]string{"component": "generation_finisher", "status": "error"}).Counter("iterations.count"),
	}
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

func (s *Scheduler) discoverServices(ctx context.Context, start, end time.Time) ([]serviceInfo, error) {
	builder := sq.Select("service", "count(*) as count").
		From("profiles").
		Where(sq.GtOrEq{"timestamp": start}).
		Where(sq.Lt{"timestamp": end}).
		GroupBy("service").
		OrderBy("count DESC").
		Limit(uint64(s.conf.MaxServices))

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build clickhouse query: %w", err)
	}

	rows, err := s.storage.DBs.ClickhouseConn.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute clickhouse query: %w", err)
	}
	defer rows.Close()

	var services []serviceInfo
	for rows.Next() {
		var info serviceInfo
		if err := rows.Scan(&info.name, &info.profileCount); err != nil {
			return nil, fmt.Errorf("failed to scan service info: %w", err)
		}
		services = append(services, info)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read clickhouse services info: %w", err)
	}

	heavyCount := int(float64(len(services)) * s.conf.HeavyPercent / 100.0)
	for i := 0; i < heavyCount; i++ {
		services[i].heavy = true
	}

	return services, nil
}

func (s *Scheduler) createGeneration(ctx context.Context, generationID int32, start, end time.Time, services []serviceInfo) error {
	primary, err := s.storage.DBs.PostgresCluster.WaitForPrimary(ctx)
	if err != nil {
		return fmt.Errorf("failed to wait for primary postgres: %w", err)
	}

	tx, err := primary.DBx().BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to start postgres tx: %w", err)
	}
	defer tx.Rollback()

	err = tx.QueryRowContext(ctx,
		`INSERT INTO cluster_top_generations (id, from_ts, to_ts, status) 
		 VALUES ($1, $2, $3, $4) 
		 ON CONFLICT (id) DO NOTHING 
		 RETURNING id`,
		generationID, start, end, generationStatusScheduled,
	).Scan(&generationID)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errGenerationAlreadyExists
		}
		return fmt.Errorf("failed to insert generation: %w", err)
	}

	for i := 0; i < len(services); i += maxServicesInsertBatchSize {
		end := min(i+maxServicesInsertBatchSize, len(services))
		batch := services[i:end]

		builder := sq.StatementBuilder.PlaceholderFormat(sq.Dollar).
			Insert("cluster_top_services").
			Columns("service", "status", "profiles_count", "generation", "heavy")

		for _, svc := range batch {
			builder = builder.Values(svc.name, "ready", svc.profileCount, generationID, svc.heavy)
		}

		query, args, err := builder.ToSql()
		if err != nil {
			return fmt.Errorf("failed to build postgres insert query: %w", err)
		}

		_, err = tx.ExecContext(ctx, query, args...)
		if err != nil {
			return fmt.Errorf("failed to insert services: %w", err)
		}
	}

	return tx.Commit()
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

	generationID := lastID + 1

	s.l.Info(ctx, "Trying to schedule new generation",
		log.Time("start", targetStart),
		log.Time("end", targetEnd),
		log.Int("generation_id", int(generationID)),
	)

	services, err := s.discoverServices(ctx, targetStart, targetEnd)
	if err != nil {
		return fmt.Errorf("failed to discover services: %w", err)
	}

	err = s.createGeneration(ctx, generationID, targetStart, targetEnd, services)
	if err != nil {
		return fmt.Errorf("failed to create generation: %w", err)
	}

	s.l.Info(ctx, "Successfully created new generation",
		log.Time("start", targetStart),
		log.Time("end", targetEnd),
		log.Int("services_count", len(services)),
	)

	return nil
}

func (s *Scheduler) finishGenerations(ctx context.Context) error {
	primary, err := s.storage.DBs.PostgresCluster.WaitForPrimary(ctx)
	if err != nil {
		return fmt.Errorf("failed to wait for primary postgres: %w", err)
	}

	var scheduledIDs []int
	query, args, err := sq.Select("id").
		From("cluster_top_generations").
		Where(sq.Eq{"status": generationStatusScheduled}).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return fmt.Errorf("failed to build in_progress query: %w", err)
	}

	err = primary.DBx().SelectContext(ctx, &scheduledIDs, query, args...)
	if err != nil {
		return fmt.Errorf("failed to fetch in_progress generations: %w", err)
	}

	errs := []error{}
	for _, id := range scheduledIDs {
		var pendingCount int
		err = primary.DBx().GetContext(ctx, &pendingCount,
			`SELECT count(*) FROM cluster_top_services WHERE generation = $1 AND status = 'ready'`,
			id,
		)
		if err != nil {
			s.l.Error(ctx, "Failed to count pending services for in_progress generation", log.Int("id", id), log.Error(err))
			errs = append(errs, err)
			continue
		}

		if pendingCount == 0 {
			_, err = primary.DBx().ExecContext(ctx,
				`UPDATE cluster_top_generations SET status = $1 WHERE id = $2`,
				generationStatusFinished, id,
			)
			if err != nil {
				s.l.Error(ctx, "Failed to update generation status to finished", log.Int("id", id), log.Error(err))
				errs = append(errs, err)
			}
		}
	}

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
