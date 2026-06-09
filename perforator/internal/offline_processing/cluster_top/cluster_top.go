package cluster_top

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/observability/lib/querylang"
	"github.com/yandex/perforator/observability/lib/querylang/operator"
	"github.com/yandex/perforator/perforator/internal/asyncfilecache"
	"github.com/yandex/perforator/perforator/internal/symbolizer/binaryprovider/downloader"
	"github.com/yandex/perforator/perforator/internal/xmetrics"
	"github.com/yandex/perforator/perforator/pkg/profilequerylang"
	"github.com/yandex/perforator/perforator/pkg/sampletype"
	"github.com/yandex/perforator/perforator/pkg/storage/bundle"
	"github.com/yandex/perforator/perforator/pkg/storage/profile"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

type ClusterTop struct {
	l xlog.Logger

	downloader *downloader.Downloader

	profileStorage profile.Storage

	symbolizer *ClusterTopSymbolizer

	metrics *workerMetrics
}

func NewClusterTop(
	conf *Config,
	l xlog.Logger,
	reg xmetrics.Registry,
	storageBundle *bundle.StorageBundle,
) (*ClusterTop, error) {
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

	gsymDownloader, err := downloader.NewGSYMDownloader(downloaderInstance, storageBundle.GSYMStorage)
	if err != nil {
		return nil, err
	}

	symbolizer, err := NewClusterTopSymbolizer(l, gsymDownloader)
	if err != nil {
		return nil, err
	}

	return &ClusterTop{
		l:              l,
		downloader:     downloaderInstance,
		profileStorage: storageBundle.ProfileStorage,
		symbolizer:     symbolizer,
		metrics:        newWorkerMetrics(reg),
	}, nil
}

func buildSelector(serviceName string, timeRange TimeRange, podID, nodeID string) (*querylang.Selector, error) {
	selectorStr := fmt.Sprintf("{%s=\"%s\", %s=\"%s\", %s=\"%s\", %s=\"%s\"}",
		profilequerylang.EventTypeLabel, sampletype.SampleTypeCPUCycles,
		profilequerylang.ServiceLabel, serviceName,
		profilequerylang.SystemNameLabel, "perforator",
		profilequerylang.CPOIDLabel, "",
	)

	selector, err := profilequerylang.ParseSelector(selectorStr)
	if err != nil {
		return nil, err
	}

	if podID != "" {
		selector.Matchers = append(
			selector.Matchers,
			profilequerylang.BuildMatcher(
				profilequerylang.PodIDLabel,
				querylang.AND,
				querylang.Condition{Operator: operator.Eq},
				[]string{podID},
			),
		)
	}

	if nodeID != "" {
		selector.Matchers = append(
			selector.Matchers,
			profilequerylang.BuildMatcher(
				profilequerylang.NodeIDLabel,
				querylang.AND,
				querylang.Condition{Operator: operator.Eq},
				[]string{nodeID},
			),
		)
	}

	selector.Matchers = append(
		selector.Matchers,
		profilequerylang.BuildMatcher(
			profilequerylang.TimestampLabel,
			querylang.AND,
			querylang.Condition{Operator: operator.GTE},
			[]string{timeRange.From.Format(time.RFC3339Nano)},
		),
	)

	selector.Matchers = append(
		selector.Matchers,
		profilequerylang.BuildMatcher(
			profilequerylang.TimestampLabel,
			querylang.AND,
			querylang.Condition{Operator: operator.LT},
			[]string{timeRange.To.Format(time.RFC3339Nano)},
		),
	)

	return selector, nil
}

const kDefaultProfilesBatchSize int = 200

func (t *ClusterTop) Run(
	ctx context.Context,
	jobSelector JobSelector,
	clusterPerfTopAggregator ClusterPerfTopAggregator,
	degreeOfParallelism uint,
) error {
	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		err := t.downloader.RunBackgroundDownloader(ctx)
		if err != nil {
			t.l.Error(ctx, "Failed background downloader", log.Error(err))
		}
		return err
	})

	g.Go(func() error {
		aggregateG, ctx := errgroup.WithContext(ctx)

		for range degreeOfParallelism {
			aggregateG.Go(func() error {
				for {
					shouldContinueRightAway := t.selectAndProcessJob(
						ctx,
						jobSelector,
						clusterPerfTopAggregator,
						int(degreeOfParallelism),
					)
					if !shouldContinueRightAway {
						if ctx.Err() != nil {
							break
						}

						time.Sleep(10 * time.Second)
					}
				}

				return nil
			})
		}

		err := aggregateG.Wait()
		if err != nil {
			return err
		}

		return nil
	})

	return g.Wait()
}

func (t *ClusterTop) selectAndProcessJob(
	ctx context.Context,
	jobSelector JobSelector,
	clusterPerfTopAggregator ClusterPerfTopAggregator,
	degreeOfParallelism int,
) (shouldContinueRightAway bool) {
	selected, err := jobSelector.SelectJob(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			t.l.Info(ctx, "No cluster top jobs")
			return false
		}
		t.l.Warn(ctx, "Failed to select a job", log.Error(err))
		return false
	}

	job := selected.Job
	processor := newOneShotJobProcessor(
		t.l,
		t.profileStorage,
		t.symbolizer,
		clusterPerfTopAggregator,
		job,
		degreeOfParallelism,
		kDefaultProfilesBatchSize,
	)

	var jobResult oneShotJobResult
	defer func() {
		stats := &jobResult.executionStats

		l := t.l.With(
			log.Int64("job_id", job.ID),
			log.String("service", job.Service),
			log.Int("generation", job.Generation),
			log.String("workload_key", job.WorkloadKey()),
			log.String("pod_id", job.PodID),
			log.String("node_id", job.NodeID),
			log.Time("from", job.TimeRange.From),
			log.Time("to", job.TimeRange.To),
			log.Int("profilesCount", jobResult.profilesProcessed),
			log.Any("execution_stats", stats),
		)
		if err != nil {
			l.Error(ctx, "Failed to process the job", log.Error(err))
		} else {
			l.Info(ctx, "Successfully processed the job")
		}

		t.metrics.recordJob(err, jobResult.profilesProcessed, stats)
		selected.Finalize(ctx, err, stats)
	}()

	jobResult, err = processor.run(ctx)

	return true
}
