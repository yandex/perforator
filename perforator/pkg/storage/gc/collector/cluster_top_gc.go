package collector

import (
	"context"
	"fmt"
	"time"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/library/go/core/metrics"
	clustertopgc "github.com/yandex/perforator/perforator/pkg/storage/cluster_top/gc"
	generation "github.com/yandex/perforator/perforator/pkg/storage/cluster_top/generations"
	"github.com/yandex/perforator/perforator/pkg/storage/gc/config"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

type clusterTopGCStorage interface {
	ListGenerations(context.Context) ([]clustertopgc.Generation, error)
	MarkDeleting(context.Context, uint32, time.Time) (bool, error)
	DeleteData(context.Context, uint32) error
	DeleteJobs(context.Context, uint32, uint32) (int64, error)
	DeleteGeneration(context.Context, uint32) error
}

type clusterTopGC struct {
	l                         xlog.Logger
	storage                   clusterTopGCStorage
	conf                      config.ClusterTopConfig
	generations, jobs, errors metrics.Counter
	candidates, deleting      metrics.Gauge
	duration                  metrics.Timer
}

func newClusterTopGC(l xlog.Logger, r metrics.Registry, conf config.ClusterTopConfig, storage clusterTopGCStorage) *clusterTopGC {
	r = r.WithPrefix("cluster_top_gc")
	return &clusterTopGC{
		l: l.WithName("cluster_top_gc"), storage: storage, conf: conf,
		generations: r.Counter("generations.deleted"), jobs: r.Counter("jobs.deleted"), errors: r.Counter("errors"),
		candidates: r.Gauge("generations.pending"), deleting: r.Gauge("generations.deleting"), duration: r.Timer("iteration.duration"),
	}
}

func selectClusterTopGenerations(generations []clustertopgc.Generation, cutoff time.Time) []clustertopgc.Generation {
	var newestFinished uint32
	for _, g := range generations {
		if g.Status == generation.StatusFinished {
			newestFinished = max(newestFinished, g.ID)
		}
	}
	var candidates []clustertopgc.Generation
	for _, g := range generations {
		if g.BucketCount == 0 || g.HasPendingJobs {
			continue
		}
		if g.Status == generation.StatusDeleting ||
			(g.Status == generation.StatusFinished && g.ID < newestFinished && g.To.Before(cutoff)) {
			candidates = append(candidates, g)
		}
	}
	return candidates
}

func waitClusterTopGC(ctx context.Context, duration time.Duration) error {
	if err := context.Cause(ctx); err != nil {
		return err
	}
	timer := time.NewTimer(max(duration, 0))
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return context.Cause(ctx)
	case <-timer.C:
		return context.Cause(ctx)
	}
}

func (c *clusterTopGC) run(ctx context.Context) error {
	for {
		if err := context.Cause(ctx); err != nil {
			return err
		}
		if err := c.collect(ctx); err != nil {
			c.l.Error(ctx, "Failed to collect Cluster Top generations", log.Error(err))
		}
		if err := waitClusterTopGC(ctx, c.conf.Interval); err != nil {
			return err
		}
	}
}

func (c *clusterTopGC) collect(ctx context.Context) (err error) {
	start := time.Now()
	defer func() {
		c.duration.RecordDuration(time.Since(start))
		if err != nil {
			c.errors.Inc()
		}
	}()
	if err := context.Cause(ctx); err != nil {
		return err
	}
	generations, err := c.storage.ListGenerations(ctx)
	if err != nil {
		return err
	}
	cutoff := time.Now().Add(-c.conf.TTL)
	candidates := selectClusterTopGenerations(generations, cutoff)
	c.candidates.Set(float64(len(candidates)))
	var deleting int
	for _, g := range candidates {
		if g.Status == generation.StatusDeleting {
			deleting++
		}
	}
	c.deleting.Set(float64(deleting))
	for _, g := range candidates {
		if err := c.collectGeneration(ctx, g, cutoff); err != nil {
			return err
		}
	}
	return nil
}

func (c *clusterTopGC) collectGeneration(ctx context.Context, g clustertopgc.Generation, cutoff time.Time) error {
	if err := context.Cause(ctx); err != nil {
		return err
	}
	marked, err := c.storage.MarkDeleting(ctx, g.ID, cutoff)
	if err != nil {
		return fmt.Errorf("mark generation %d: %w", g.ID, err)
	}
	if !marked {
		c.candidates.Add(-1)
		return nil
	}
	if g.Status != generation.StatusDeleting {
		c.deleting.Add(1)
	}
	c.l.Info(ctx, "Collecting Cluster Top generation", log.UInt32("generation", g.ID))
	if err := context.Cause(ctx); err != nil {
		return err
	}
	if err := c.storage.DeleteData(ctx, g.ID); err != nil {
		return err
	}
	for {
		if err := context.Cause(ctx); err != nil {
			return err
		}
		count, err := c.storage.DeleteJobs(ctx, g.ID, c.conf.JobsBatchSize)
		if err != nil {
			return fmt.Errorf("delete jobs for generation %d: %w", g.ID, err)
		}
		c.jobs.Add(count)
		if count == 0 {
			break
		}
		if err := waitClusterTopGC(ctx, c.conf.JobsInterval); err != nil {
			return err
		}
	}
	if err := context.Cause(ctx); err != nil {
		return err
	}
	if err := c.storage.DeleteGeneration(ctx, g.ID); err != nil {
		return err
	}
	c.generations.Inc()
	c.candidates.Add(-1)
	c.deleting.Add(-1)
	return nil
}
