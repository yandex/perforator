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

func buildSelector(serviceName string, timeRange TimeRange) (*querylang.Selector, error) {
	selectorStr := fmt.Sprintf("{%s=\"%s\", %s=\"%s\", %s=\"%s\"}",
		profilequerylang.EventTypeLabel, sampletype.SampleTypeCPUCycles,
		profilequerylang.ServiceLabel, serviceName,
		profilequerylang.SystemNameLabel, "perforator",
	)

	selector, err := profilequerylang.ParseSelector(selectorStr)
	if err != nil {
		return nil, err
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
const kHeavyProfilesBatchSize int = 50

func (t *ClusterTop) Run(
	ctx context.Context,
	serviceSelector ServiceSelector,
	clusterPerfTopAggregator ClusterPerfTopAggregator,
	heavy bool,
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

		servicesDegreeOfParallelism := int(degreeOfParallelism)
		profilesDegreeOfParallelism := 1
		if heavy {
			servicesDegreeOfParallelism = 1
			profilesDegreeOfParallelism = int(degreeOfParallelism)
		}

		for range servicesDegreeOfParallelism {
			aggregateG.Go(func() error {
				for {
					shouldContinueRightAway := t.selectAndProcessService(
						ctx,
						serviceSelector,
						clusterPerfTopAggregator,
						heavy,
						profilesDegreeOfParallelism,
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

func (t *ClusterTop) selectAndProcessService(
	ctx context.Context,
	serviceSelector ServiceSelector,
	clusterPerfTopAggregator ClusterPerfTopAggregator,
	heavy bool,
	degreeOfParallelism int,
) (shouldContinueRightAway bool) {
	serviceHandler, err := serviceSelector.SelectService(ctx, heavy)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			t.l.Info(ctx, "No cluster top service jobs")
			return false
		}
		t.l.Warn(ctx, "Failed to select a service", log.Error(err))
		// In case of select service failure - we should avoid retrying immediately in the upper layer
		return false
	}

	startTime := time.Now()
	var profilesCount int
	defer func() {
		duration := time.Since(startTime)
		l := t.l.With(
			log.String("service", serviceHandler.GetServiceName()),
			log.Duration("duration", duration),
			log.Int("generation", serviceHandler.GetGeneration()),
			log.Time("from", serviceHandler.GetTimeRange().From),
			log.Time("to", serviceHandler.GetTimeRange().To),
			log.Int("profilesCount", profilesCount),
		)
		if err != nil {
			l.Error(ctx, "Failed to process the service", log.Error(err))
		} else {
			l.Info(ctx, "Successfully processed the service")
		}
		serviceHandler.Finalize(ctx, err)
	}()

	profilesBatchSize := kDefaultProfilesBatchSize
	if heavy {
		profilesBatchSize = kHeavyProfilesBatchSize
	}

	profilesCount, err = t.processService(
		ctx,
		clusterPerfTopAggregator,
		serviceHandler.GetGeneration(),
		serviceHandler.GetServiceName(),
		serviceHandler.GetTimeRange(),
		degreeOfParallelism,
		profilesBatchSize,
	)

	return true
}

func (t *ClusterTop) processService(
	ctx context.Context,
	clusterPerfTopAggregator ClusterPerfTopAggregator,
	generation int,
	serviceName string,
	timeRange TimeRange,
	degreeOfParallelism int,
	profilesBatchSize int,
) (processedProfiles int, err error) {
	selector, err := buildSelector(serviceName, timeRange)
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
		t.l.Warn(ctx, "Service has no profiles, marking as done",
			log.String("service", serviceName),
		)
		return 0, nil
	}

	t.l.Info(ctx, "Starting service processing",
		log.String("service", serviceName),
		log.Int("profilesCount", len(profileMetas)),
	)

	buildIDs := getBuildIDsFromProfiles(profileMetas)

	functions, err := t.processServiceProfiles(
		ctx,
		serviceName,
		profileMetas,
		buildIDs,
		degreeOfParallelism,
		profilesBatchSize,
	)
	if err != nil {
		return len(profileMetas), err
	}

	err = clusterPerfTopAggregator.Save(ctx, &ServicePerfTop{
		Generation:  generation,
		ServiceName: serviceName,
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
			noExistErr := &blob.ErrNoExist{}

			data, err := t.profileStorage.FetchProfile(ctx, profileMetas[i])
			if err != nil && !errors.As(err, &noExistErr) {
				return err
			}

			profiles[i] = data

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
