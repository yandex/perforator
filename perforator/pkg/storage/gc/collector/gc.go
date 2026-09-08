package collector

import (
	"context"
	"fmt"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/yandex/perforator/library/go/core/metrics"
	"github.com/yandex/perforator/perforator/pkg/storage/bundle"
	"github.com/yandex/perforator/perforator/pkg/storage/gc/config"
	"github.com/yandex/perforator/perforator/pkg/storage/storage"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

type GC struct {
	collectors []*storageGC
}

func NewGC(l xlog.Logger, r metrics.Registry, gcConf config.Config, storageBundle *bundle.StorageBundle) (*GC, error) {
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

	return &GC{
		collectors: collectors,
	}, nil
}

func (g *GC) Run(ctx context.Context, interval time.Duration) error {
	gr, ctx := errgroup.WithContext(ctx)

	for _, collector := range g.collectors {
		gr.Go(func() error {
			return collector.run(ctx, interval)
		})
	}

	return gr.Wait()
}
