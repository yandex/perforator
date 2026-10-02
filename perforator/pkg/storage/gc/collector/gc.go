package collector

import (
	"context"
	"fmt"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/yandex/perforator/library/go/core/metrics"
	"github.com/yandex/perforator/perforator/pkg/lease"
	postgreslease "github.com/yandex/perforator/perforator/pkg/lease/postgres"
	"github.com/yandex/perforator/perforator/pkg/storage/bundle"
	clustertopgc "github.com/yandex/perforator/perforator/pkg/storage/cluster_top/gc"
	"github.com/yandex/perforator/perforator/pkg/storage/gc/config"
	"github.com/yandex/perforator/perforator/pkg/storage/storage"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

type GC struct {
	clusterTop *clusterTopGC
	collectors []*storageGC
	l          xlog.Logger
	registry   metrics.Registry
	target     lease.Lease
	leaseTTL   time.Duration
}

func NewGC(l xlog.Logger, r metrics.Registry, gcConf config.Config, storageBundle *bundle.StorageBundle) (*GC, error) {
	gcConf.FillDefault()
	if err := gcConf.Validate(); err != nil {
		return nil, err
	}

	collectors := make([]*storageGC, 0, len(gcConf.Storages))
	for _, conf := range gcConf.Storages {
		var st storage.Storage
		switch conf.Type {
		case config.Profile:
			st = storageBundle.ProfileStorage
		case config.Binary:
			st = storageBundle.BinaryStorage
		case config.GSYM:
			st = storageBundle.GSYMStorage
		default:
			return nil, fmt.Errorf("unsupported storage type %s", conf.Type)
		}
		collectors = append(collectors, newStorageGC(l, r, conf, st))
	}

	var clusterTop *clusterTopGC
	if gcConf.ClusterTop.Enabled {
		if storageBundle.LeaseStorage == nil {
			return nil, fmt.Errorf("Cluster Top GC requires shared lease storage")
		}
		if storageBundle.DBs == nil || storageBundle.DBs.PostgresCluster == nil || storageBundle.DBs.ClickhouseConn == nil {
			return nil, fmt.Errorf("Cluster Top GC requires PostgreSQL and ClickHouse")
		}
		clusterTop = newClusterTopGC(l, r, gcConf.ClusterTop, clustertopgc.NewStorage(storageBundle.DBs.PostgresCluster, storageBundle.DBs.ClickhouseConn, gcConf.ClusterTop.OperationTimeout))
	}
	var target lease.Lease
	if storageBundle.LeaseStorage != nil {
		target = postgreslease.ForKey(storageBundle.LeaseStorage, gcConf.LeaseName)
	}
	var leaseRegistry metrics.Registry
	if r != nil {
		leaseRegistry = r.WithTags(map[string]string{"name": gcConf.LeaseName})
	}
	return &GC{
		clusterTop: clusterTop,
		collectors: collectors,
		l:          l,
		registry:   leaseRegistry,
		target:     target,
		leaseTTL:   gcConf.LeaseTTL,
	}, nil
}

func (g *GC) Run(ctx context.Context, interval time.Duration) error {
	if interval <= 0 {
		return fmt.Errorf("GC iteration interval must be positive")
	}
	// Keep custom configurations without lease storage backwards-compatible.
	if g.target == nil {
		return g.runCollectors(ctx, interval)
	}
	var runErr error
	err := lease.LockAndRun(ctx, g.l, g.target,
		func(leaseCtx context.Context, _ string) {
			runErr = g.runCollectors(leaseCtx, interval)
			if cause := context.Cause(leaseCtx); cause != nil {
				runErr = cause
			}
		}, lease.WithTTL(g.leaseTTL), lease.WithMetrics(g.registry))
	if err != nil {
		return err
	}
	return runErr
}

func (g *GC) runCollectors(ctx context.Context, interval time.Duration) error {
	gr, ctx := errgroup.WithContext(ctx)

	for _, collector := range g.collectors {
		gr.Go(func() error {
			return collector.run(ctx, interval)
		})
	}

	if g.clusterTop != nil {
		gr.Go(func() error { return g.clusterTop.run(ctx) })
	}
	return gr.Wait()
}
