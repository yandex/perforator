package scheduler

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	hasql "golang.yandex/hasql/sqlx"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/library/go/core/metrics"
	"github.com/yandex/perforator/perforator/pkg/storage/cluster_top/aggregated"
	"github.com/yandex/perforator/perforator/pkg/storage/cluster_top/generations"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

type ServingTablesConfig struct {
	RefreshInterval time.Duration             `yaml:"refresh_interval"`
	Rebuild         aggregated.SnapshotConfig `yaml:"rebuild"`
}

func (c *ServingTablesConfig) FillDefault() {
	if c.RefreshInterval == 0 {
		c.RefreshInterval = time.Hour
	}
	c.Rebuild.FillDefault()
}

func (c *ServingTablesConfig) Validate() error {
	if c.RefreshInterval <= 0 {
		return fmt.Errorf("serving refresh interval must be positive")
	}
	return c.Rebuild.Validate()
}

type snapshotBuilder interface {
	RebuildServingSnapshot(context.Context, GenerationID, aggregated.SnapshotConfig) error
}

type GenerationController struct {
	l            xlog.Logger
	cluster      *hasql.Cluster
	conf         ServingTablesConfig
	snapshots    snapshotBuilder
	lastSnapshot map[GenerationID]time.Time

	iterationSuccess          metrics.Counter
	iterationErrors           metrics.Counter
	pendingJobsGauge          metrics.IntGauge
	scheduledGenerationsGauge metrics.IntGauge
	snapshotSuccess           metrics.Counter
	snapshotErrors            metrics.Counter
	snapshotDuration          metrics.Timer
	snapshotAge               metrics.GaugeVec
}

func newGenerationController(
	l xlog.Logger,
	r metrics.Registry,
	cluster *hasql.Cluster,
	snapshots snapshotBuilder,
	conf ServingTablesConfig,
) *GenerationController {
	return &GenerationController{
		l:                         l.WithName("GenerationController"),
		cluster:                   cluster,
		conf:                      conf,
		snapshots:                 snapshots,
		lastSnapshot:              make(map[GenerationID]time.Time),
		iterationSuccess:          r.WithTags(map[string]string{"component": "generation_controller", "status": "success"}).Counter("iterations.count"),
		iterationErrors:           r.WithTags(map[string]string{"component": "generation_controller", "status": "error"}).Counter("iterations.count"),
		pendingJobsGauge:          r.IntGauge("jobs.pending.count"),
		scheduledGenerationsGauge: r.IntGauge("generations.scheduled.count"),
		snapshotSuccess:           r.WithTags(map[string]string{"outcome": "success"}).Counter("serving.rebuilds.count"),
		snapshotErrors:            r.WithTags(map[string]string{"outcome": "error"}).Counter("serving.rebuilds.count"),
		snapshotDuration:          r.Timer("serving.rebuild.duration"),
		snapshotAge:               r.GaugeVec("serving.snapshot.age_seconds", []string{"generation"}),
	}
}

// Run processes generation transitions and rebuilds serially under the scheduler lease.
func (s *GenerationController) Run(ctx context.Context) error {
	clear(s.lastSnapshot)
	ticker := time.NewTicker(30 * time.Second)

	iteration := func() {
		if err := s.processGenerations(ctx); err != nil {
			s.l.Error(ctx, "Failed to run generation controller tick", log.Error(err))
			s.iterationErrors.Inc()
		} else {
			s.iterationSuccess.Inc()
		}
	}
	iteration()
	for {
		select {
		case <-ctx.Done():
			return context.Cause(ctx)
		case <-ticker.C:
			iteration()
		}
	}
}

type activeGeneration struct {
	ID                  GenerationID `db:"id"`
	Status              string       `db:"status"`
	UnfinishedJobsCount int64        `db:"unfinished_jobs_count"`
}

func (s *GenerationController) getActiveGenerations(ctx context.Context) ([]activeGeneration, error) {
	primary, err := s.cluster.WaitForPrimary(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to wait for primary postgres: %w", err)
	}
	var active []activeGeneration
	err = primary.DBx().SelectContext(ctx, &active, `
        SELECT g.id, g.status,
               (SELECT count(*) FROM cluster_top_jobs j WHERE j.generation = g.id
                AND j.status IN ('pending', 'running')) AS unfinished_jobs_count
        FROM cluster_top_generations g WHERE g.status IN ($1, $2) ORDER BY g.id`,
		generations.StatusScheduled, generations.StatusFinalizing)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch active generations: %w", err)
	}
	return active, nil
}

func (s *GenerationController) processGenerations(ctx context.Context) error {
	active, err := s.getActiveGenerations(ctx)
	if err != nil {
		return err
	}
	s.snapshotAge.Reset()
	var errs []error
	var totalUnfinishedJobsCount int64
	remainingCount := len(active)
	for _, generation := range active {
		if err := context.Cause(ctx); err != nil {
			return err
		}
		totalUnfinishedJobsCount += generation.UnfinishedJobsCount
		finished, err := s.processGeneration(ctx, generation)
		if err != nil {
			errs = append(errs, fmt.Errorf("process generation %d: %w", generation.ID, err))
		} else if finished {
			remainingCount--
			delete(s.lastSnapshot, generation.ID)
		}
	}
	s.pendingJobsGauge.Set(totalUnfinishedJobsCount)
	s.scheduledGenerationsGauge.Set(int64(remainingCount))
	return errors.Join(errs...)
}

func (s *GenerationController) processGeneration(ctx context.Context, generation activeGeneration) (bool, error) {
	if generation.UnfinishedJobsCount != 0 {
		if generation.Status == generations.StatusFinalizing {
			return false, fmt.Errorf("finalizing generation still has unfinished jobs")
		}
		return false, s.refreshSnapshot(ctx, generation.ID, false)
	}
	if generation.Status == generations.StatusScheduled {
		changed, err := s.transitionGeneration(ctx, generation.ID, generations.StatusScheduled, generations.StatusFinalizing)
		if err != nil || !changed {
			return false, err
		}
	}
	if err := s.refreshSnapshot(ctx, generation.ID, true); err != nil {
		return false, err
	}
	return s.transitionGeneration(ctx, generation.ID, generations.StatusFinalizing, generations.StatusFinished)
}

func (s *GenerationController) transitionGeneration(ctx context.Context, id GenerationID, from, to string) (bool, error) {
	primary, err := s.cluster.WaitForPrimary(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to wait for primary postgres: %w", err)
	}
	result, err := primary.DBx().ExecContext(ctx, `
        UPDATE cluster_top_generations g SET status = $1
        WHERE g.id = $2 AND g.status = $3
          AND NOT EXISTS (SELECT 1 FROM cluster_top_jobs j WHERE j.generation = g.id
                          AND j.status IN ('pending', 'running'))`, to, id, from)
	if err != nil {
		return false, fmt.Errorf("set status %s: %w", to, err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("read status %s update result: %w", to, err)
	}
	return count != 0, nil
}

func (s *GenerationController) refreshSnapshot(ctx context.Context, generation GenerationID, final bool) error {
	last := s.lastSnapshot[generation]
	age := s.snapshotAge.With(map[string]string{"generation": strconv.FormatUint(uint64(generation), 10)})
	if last.IsZero() {
		age.Set(-1) // no successful snapshot in this leadership tenure
	} else {
		age.Set(time.Since(last).Seconds())
		if !final && time.Since(last) < s.conf.RefreshInterval {
			return nil
		}
	}
	started := time.Now()
	err := s.snapshots.RebuildServingSnapshot(ctx, generation, s.conf.Rebuild)
	s.snapshotDuration.RecordDuration(time.Since(started))
	if err != nil {
		s.snapshotErrors.Inc()
		s.l.Error(ctx, "Failed to rebuild serving snapshot", log.UInt32("generation", generation), log.Bool("final", final), log.Error(err))
		return err
	}
	s.lastSnapshot[generation] = time.Now()
	age.Set(0)
	s.snapshotSuccess.Inc()
	s.l.Info(ctx, "Published serving snapshot", log.UInt32("generation", generation), log.Bool("final", final), log.Duration("duration", time.Since(started)))
	return nil
}
