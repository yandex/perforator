package collector

import (
	"context"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/yandex/perforator/library/go/core/metrics"
	"github.com/yandex/perforator/perforator/pkg/storage/bundle"
	"github.com/yandex/perforator/perforator/pkg/storage/gc/config"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

type GC struct {
	collectors []Collector
}

func NewGC(l xlog.Logger, r metrics.Registry, gcConf config.Config, storageBundle *bundle.StorageBundle) (*GC, error) {
	gcConf.FillDefault()

	collectors := make([]Collector, 0, len(gcConf.Storages))
	for _, conf := range gcConf.Storages {
		collector, err := NewCollector(l, r, &conf, storageBundle)
		if err != nil {
			return nil, err
		}
		collectors = append(collectors, collector)
	}

	return &GC{
		collectors: collectors,
	}, nil
}

func (g *GC) Run(ctx context.Context, interval time.Duration) error {
	gr, ctx := errgroup.WithContext(ctx)

	for _, collector := range g.collectors {
		collectorCopy := collector
		gr.Go(func() error {
			return collectorCopy.Run(ctx, interval)
		})
	}

	return gr.Wait()
}
