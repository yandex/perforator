package cluster_top

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync/atomic"
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
	blob "github.com/yandex/perforator/perforator/pkg/storage/blob/models"
	"github.com/yandex/perforator/perforator/pkg/storage/bundle"
	"github.com/yandex/perforator/perforator/pkg/storage/profile"
	"github.com/yandex/perforator/perforator/pkg/storage/profile/meta"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

type ClusterTop struct {
	l xlog.Logger

	downloader *downloader.Downloader

	profileStorage profile.Storage

	symbolizer *ClusterTopSymbolizer
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
		l: l,

		downloader: downloaderInstance,

		profileStorage: storageBundle.ProfileStorage,

		symbolizer: symbolizer,
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
	startTime := time.Now()
	var profilesCount int
	defer func() {
		duration := time.Since(startTime)
		l := t.l.With(
			log.Int64("job_id", job.ID),
			log.String("service", job.Service),
			log.Duration("duration", duration),
			log.Int("generation", job.Generation),
			log.String("workload_key", job.WorkloadKey()),
			log.String("pod_id", job.PodID),
			log.String("node_id", job.NodeID),
			log.Time("from", job.TimeRange.From),
			log.Time("to", job.TimeRange.To),
			log.Int("profilesCount", profilesCount),
		)
		if err != nil {
			l.Error(ctx, "Failed to process the job", log.Error(err))
		} else {
			l.Info(ctx, "Successfully processed the job")
		}
		selected.Finalize(ctx, err)
	}()

	profilesCount, err = t.processJob(
		ctx,
		clusterPerfTopAggregator,
		job,
		degreeOfParallelism,
		kDefaultProfilesBatchSize,
	)

	return true
}

func (t *ClusterTop) processJob(
	ctx context.Context,
	clusterPerfTopAggregator ClusterPerfTopAggregator,
	job Job,
	degreeOfParallelism int,
	profilesBatchSize int,
) (processedProfiles int, err error) {
	selector, err := buildSelector(job.Service, job.TimeRange, job.PodID, job.NodeID)
	if err != nil {
		return 0, err
	}

	profileMetas, err := t.profileStorage.SelectProfiles(ctx, &meta.ProfileQuery{
		Selector: selector,
	})
	if err != nil {
		return 0, err
	}

	if len(profileMetas) == 0 {
		t.l.Warn(ctx, "Job has no profiles, marking as done",
			log.String("service", job.Service),
			log.String("workload_key", job.WorkloadKey()),
		)
		return 0, nil
	}

	t.l.Info(ctx, "Starting job processing",
		log.String("service", job.Service),
		log.String("workload_key", job.WorkloadKey()),
		log.String("pod_id", job.PodID),
		log.String("node_id", job.NodeID),
		log.Int("profilesCount", len(profileMetas)),
	)

	buildIDs := getBuildIDsFromProfiles(profileMetas)

	functions, err := t.processServiceProfiles(
		ctx,
		job.Service,
		profileMetas,
		buildIDs,
		degreeOfParallelism,
		profilesBatchSize,
	)
	if err != nil {
		return len(profileMetas), err
	}

	err = clusterPerfTopAggregator.Save(ctx, &ServicePerfTop{
		Generation:  job.Generation,
		ServiceName: job.Service,
		Functions:   functions,
	})
	return len(profileMetas), err
}

func (t *ClusterTop) processServiceProfiles(
	ctx context.Context,
	serviceName string,
	profileMetas []*meta.ProfileMetadata,
	buildIDs []string,
	degreeOfParallelism int,
	profilesBatchSize int,
) ([]Function, error) {
	metaBatchesChan := make(
		chan []*meta.ProfileMetadata,
		// round up to make all the batches fit
		(len(profileMetas)+profilesBatchSize-1)/profilesBatchSize,
	)
	for i := 0; i < len(profileMetas); i += profilesBatchSize {
		metaBatchesChan <- profileMetas[i:min(i+profilesBatchSize, len(profileMetas))]
	}
	close(metaBatchesChan)

	gsyms, err := t.symbolizer.DownloadAllGSYMs(ctx, buildIDs)
	if err != nil {
		return nil, err
	}
	defer gsyms.Release()

	aggregators := make([]*ServicePerfTopAggregator, degreeOfParallelism)
	defer func() {
		for _, aggregator := range aggregators {
			if aggregator != nil {
				aggregator.Destroy()
			}
		}
	}()

	processedProfiles := atomic.Int64{}

	g, ctx := errgroup.WithContext(ctx)
	for i := range degreeOfParallelism {
		g.Go(func() error {
			aggregator, err := t.symbolizer.NewServicePerfTopAggregator(serviceName)
			if err != nil {
				return err
			}
			aggregators[i] = aggregator

			aggregator.InitializeSymbolizersWithGSYMs(gsyms, buildIDs)

			for metaBatch := range metaBatchesChan {
				batch, err := t.fetchProfiles(ctx, metaBatch)
				if err != nil {
					return err
				}

				t.l.Info(
					ctx,
					"Got a batch of profiles to process",
					log.String("service", serviceName),
					log.Int("batchSize", len(batch)),
					log.Int("alreadyProcessedPct", int(processedProfiles.Load()*100/int64(len(profileMetas)))),
				)

				err = aggregator.AddProfiles(ctx, batch)
				if err != nil {
					return err
				}
				processedProfiles.Add(int64(len(batch)))
			}

			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}

	for i := 1; i < len(aggregators); i += 1 {
		aggregators[0].MergeWith(aggregators[i])
	}
	return aggregators[0].Extract(), nil
}

func (t *ClusterTop) fetchProfiles(
	ctx context.Context,
	profileMetas []*meta.ProfileMetadata,
) ([]profile.ProfileData, error) {
	profiles := make([]profile.ProfileData, len(profileMetas))

	g, ctx := errgroup.WithContext(ctx)
	for i := range profileMetas {
		g.Go(func() error {
			profileBundle, err := t.profileStorage.FetchProfile(ctx, profileMetas[i])
			if err != nil {
				noExistErr := &blob.ErrNoExist{}
				if errors.As(err, &noExistErr) {
					return nil
				}
				return err
			}

			pprofData, err := profileBundle.GetOrConvertPprof()
			if err != nil {
				return err
			}
			profiles[i] = pprofData

			return nil
		})
	}

	err := g.Wait()
	if err != nil {
		return nil, err
	}

	return profiles, nil
}

func getBuildIDsFromProfiles(profileMetas []*meta.ProfileMetadata) []string {
	uniqueBuildIDs := make(map[string]struct{})

	for _, profileMeta := range profileMetas {
		for _, buildID := range profileMeta.BuildIDs {
			uniqueBuildIDs[buildID] = struct{}{}
		}
	}

	uniqueBuildIDsList := make([]string, 0, len(uniqueBuildIDs))
	for buildID := range uniqueBuildIDs {
		uniqueBuildIDsList = append(uniqueBuildIDsList, buildID)
	}

	return uniqueBuildIDsList
}
