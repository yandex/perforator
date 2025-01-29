package app

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/perforator/internal/asyncfilecache"
	"github.com/yandex/perforator/perforator/internal/offline_processing/processors"
	"github.com/yandex/perforator/perforator/internal/symbolizer/binaryprovider/downloader"
	"github.com/yandex/perforator/perforator/internal/xmetrics"
	"github.com/yandex/perforator/perforator/pkg/storage/bundle"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

type OfflineProcessingApp struct {
	l   xlog.Logger
	reg xmetrics.Registry

	downloader *downloader.Downloader

	binarySelector BinarySelector

	processingLoop *ProcessingLoop
}

func NewOfflineProcessingApp(
	conf *Config,
	l xlog.Logger,
	reg xmetrics.Registry,
) (*OfflineProcessingApp, error) {
	ctx := context.Background()

	initCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	storageBundle, err := bundle.NewStorageBundle(initCtx, l, reg, &conf.StorageConfig)
	if err != nil {
		return nil, err
	}
	l.Info(ctx, "Initialized storage bundle")

	fileCache, err := asyncfilecache.NewFileCache(
		conf.BinaryProvider.FileCache,
		l,
		reg,
	)
	if err != nil {
		return nil, err
	}

	downloaderInstance, err := downloader.NewDownloader(
		l.WithName("Downloader"),
		reg,
		fileCache,
		downloader.Config{
			MaxSimultaneousDownloads: uint64(conf.BinaryProvider.MaxSimultaneousDownloads),
		},
	)
	if err != nil {
		return nil, err
	}

	binaryDownloader, err := downloader.NewBinaryDownloader(downloaderInstance, storageBundle.BinaryStorage.Binary())
	if err != nil {
		return nil, err
	}

	binarySelector, err := NewPgBinarySelector(
		l.WithName("pg_binary_selector"),
		storageBundle.DBs.PostgresCluster,
	)
	if err != nil {
		return nil, err
	}

	binaryFetcher, err := NewS3BinaryFetcher(binaryDownloader)
	if err != nil {
		return nil, err
	}

	gsymProcessor, err := processors.NewGsymProcessor(
		l,
		reg,
		storageBundle.DBs.S3Client,
		conf.GsymS3Bucket,
	)
	if err != nil {
		return nil, err
	}

	processingLoop, err := NewProcessingLoop(
		l.WithName("ProcessingLoop"),
		reg.WithPrefix("binary_processing"),
		binarySelector,
		binaryFetcher,
		[]BinaryProcessor{
			gsymProcessor,
		},
	)
	if err != nil {
		return nil, err
	}

	return &OfflineProcessingApp{
		l:              l,
		reg:            reg,
		downloader:     downloaderInstance,
		binarySelector: binarySelector,
		processingLoop: processingLoop,
	}, nil
}

func (a *OfflineProcessingApp) Run(ctx context.Context) error {
	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		err := a.downloader.RunBackgroundDownloader(context.Background())
		if err != nil {
			a.l.Error(ctx, "Failed background downloader", log.Error(err))
		}
		return err
	})

	g.Go(func() error {
		return a.processingLoop.Run(ctx)
	})

	g.Go(func() error {
		queueSizeMetric := a.reg.IntGauge("binary_queue_size")

		tick := time.NewTicker(time.Second * 30)
		for {
			select {
			case <-ctx.Done():
				return nil
			case <-tick.C:
			}

			queuedBinariesCount, err := a.binarySelector.GetQueuedBinariesCount(ctx)
			if err != nil {
				a.l.Warn(ctx, "Failed to acquire queue size: %w", log.Error(err))
			}

			queueSizeMetric.Set(int64(queuedBinariesCount))
		}
	})

	g.Go(func() error {
		return a.runMetricsServer(ctx, 11235)
	})

	return g.Wait()
}

func (a *OfflineProcessingApp) runMetricsServer(ctx context.Context, port uint32) error {
	a.l.Info(ctx, "Starting metrics server", log.UInt32("port", port))
	http.Handle("/metrics", a.reg.HTTPHandler(ctx, a.l))

	return http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
}
