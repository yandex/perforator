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
	"github.com/yandex/perforator/perforator/internal/symbolizer/binaryprovider/downloader"
	"github.com/yandex/perforator/perforator/internal/xmetrics"
	"github.com/yandex/perforator/perforator/pkg/filecache"
	"github.com/yandex/perforator/perforator/pkg/lease"
	"github.com/yandex/perforator/perforator/pkg/profilequerylang"
	"github.com/yandex/perforator/perforator/pkg/sampletype"
	"github.com/yandex/perforator/perforator/pkg/storage/bundle"
	"github.com/yandex/perforator/perforator/pkg/storage/profile"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

type ClusterTop struct {
	l xlog.Logger

	profileStorage profile.Storage

	symbolizer *ClusterTopSymbolizer

	skiplist *ServiceSkipList

	metrics   *workerMetrics
	leaseConf JobLeaseConfig
}

func NewClusterTop(
	conf *Config,
	l xlog.Logger,
	reg xmetrics.Registry,
	storageBundle *bundle.StorageBundle,
) (*ClusterTop, error) {
	if err := conf.Worker.Lease.Validate(); err != nil {
		return nil, err
	}
	fileCache, err := filecache.NewFileCache(conf.BinaryProvider.FileCache, reg)
	if err != nil {
		return nil, err
	}

	downloaderInstance := downloader.NewDownloader(
		l.WithName("Downloader"),
		reg,
		fileCache,
		downloader.Config{
			MaxSimultaneousDownloads: uint64(conf.BinaryProvider.MaxSimultaneousDownloads),
		},
	)

	gsymDownloader := downloader.NewGSYMDownloader(downloaderInstance, storageBundle.GSYMStorage)

	symbolizer, err := NewClusterTopSymbolizer(l, gsymDownloader)
	if err != nil {
		return nil, err
	}

	return &ClusterTop{
		l:              l,
		profileStorage: storageBundle.ProfileStorage,
		symbolizer:     symbolizer,
		skiplist:       NewServiceSkipList(conf.Worker.SkippedServices),
		metrics:        newWorkerMetrics(reg),
		leaseConf:      conf.Worker.Lease,
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
	if degreeOfParallelism == 0 {
		return fmt.Errorf("cluster top parallelism must be positive")
	}
	g, ctx := errgroup.WithContext(ctx)

	for range degreeOfParallelism {
		g.Go(func() error {
			for ctx.Err() == nil {
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

					select {
					case <-ctx.Done():
					case <-time.After(10 * time.Second):
					}
				}
			}

			return nil
		})
	}

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
	if t.skiplist.Contains(job.Service) {
		err = t.runSelectedJob(ctx, selected, func(jobCtx context.Context, token string) error {
			stats := buildSkippedStats(selected.Job)
			if err := t.finalizeJob(jobCtx, selected, token, JobStatusSkipped, stats); err != nil {
				return errors.Join(err, context.Cause(jobCtx))
			}
			t.jobLogger(job).Info(ctx, "Skipping service on skip list")
			t.metrics.recordSkipped(stats)
			return nil
		})
	} else {
		err = t.processSelectedJob(ctx, selected, func(jobCtx context.Context) (oneShotJobResult, error) {
			processor := newOneShotJobProcessor(t.l, t.profileStorage, t.symbolizer, selected.Job,
				degreeOfParallelism, kDefaultProfilesBatchSize)
			return processor.run(jobCtx)
		}, clusterPerfTopAggregator)
	}
	// Contention is normal: select another candidate immediately. Other errors
	// use the worker loop's existing delay instead of repeatedly hitting the DB.
	if err != nil && !errors.Is(err, lease.ErrLeaseBusy) {
		t.jobLogger(job).Warn(ctx, "Failed to run selected job", log.Error(err))
		return false
	}
	return true
}

// compute must return success only after every required processing stage has
// finished. A failed computation must never publish even a non-nil partial top.
func (t *ClusterTop) processSelectedJob(
	ctx context.Context,
	selected *SelectedJob,
	compute func(context.Context) (oneShotJobResult, error),
	aggregator ClusterPerfTopAggregator,
) error {
	return t.runSelectedJob(ctx, selected, func(jobCtx context.Context, token string) error {
		l := t.jobLogger(selected.Job)
		start := time.Now()
		result, err := compute(jobCtx)
		if err == nil && result.top != nil && jobCtx.Err() == nil {
			saveStart := time.Now()
			err = aggregator.Save(jobCtx, result.top)
			result.executionStats.Stages.SaveTop = time.Since(saveStart)
		}
		result.executionStats.Duration = time.Since(start)
		if err != nil {
			result.executionStats.Error = err.Error()
		}

		if jobCtx.Err() != nil {
			l.Info(ctx, "Job processing interrupted", log.Error(context.Cause(jobCtx)))
			return context.Cause(jobCtx)
		}
		status := JobStatusDone
		if err != nil {
			status = JobStatusFailed
			l.Error(ctx, "Failed to process job", log.Error(err))
		}
		if err := t.finalizeJob(jobCtx, selected, token, status, &result.executionStats); err != nil {
			return errors.Join(err, context.Cause(jobCtx))
		} else {
			t.metrics.recordJob(status, result.profilesProcessed, &result.executionStats)
			l.Info(ctx, "Finalized job", log.String("status", status), log.Any("execution_stats", result.executionStats))
		}
		return nil
	})
}

func (t *ClusterTop) finalizeJob(ctx context.Context, selected *SelectedJob, token, status string, stats *JobExecutionStats) error {
	if err := selected.Finalize(ctx, token, status, stats); err != nil {
		t.metrics.finalizationErrors.Inc()
		t.jobLogger(selected.Job).Error(ctx, "Failed to finalize job", log.Error(err))
		return err
	}
	return nil
}

func (t *ClusterTop) jobLogger(job Job) xlog.Logger {
	return t.l.With(
		log.Int64("job_id", job.ID),
		log.String("service", job.Service),
		log.Int("generation", job.Generation),
		log.String("workload_key", job.WorkloadKey()),
		log.String("pod_id", job.PodID),
		log.String("node_id", job.NodeID),
		log.Time("from", job.TimeRange.From),
		log.Time("to", job.TimeRange.To),
	)
}

func buildSkippedStats(job Job) *JobExecutionStats {
	stats := newJobExecutionStats()
	if !job.StartedAt.IsZero() && !job.CreatedAt.IsZero() {
		stats.QueueWait = job.StartedAt.Sub(job.CreatedAt)
	}
	stats.Duration = time.Since(job.StartedAt)
	return stats
}
