package cluster_top

import (
	"context"
	"errors"
	"sync/atomic"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/yandex/perforator/library/go/core/log"
	blob "github.com/yandex/perforator/perforator/pkg/storage/blob/models"
	"github.com/yandex/perforator/perforator/pkg/storage/profile"
	"github.com/yandex/perforator/perforator/pkg/storage/profile/meta"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

type oneShotJobProcessor struct {
	l              xlog.Logger
	profileStorage profile.Storage
	symbolizer     *ClusterTopSymbolizer
	aggregator     ClusterPerfTopAggregator

	job                 Job
	stats               *JobExecutionStats
	degreeOfParallelism int
	profilesBatchSize   int

	startedAt time.Time
}

func newOneShotJobProcessor(
	l xlog.Logger,
	profileStorage profile.Storage,
	symbolizer *ClusterTopSymbolizer,
	aggregator ClusterPerfTopAggregator,
	job Job,
	degreeOfParallelism int,
	profilesBatchSize int,
) *oneShotJobProcessor {
	stats := newJobExecutionStats()
	if !job.CreatedAt.IsZero() && !job.StartedAt.IsZero() {
		stats.QueueWait = job.StartedAt.Sub(job.CreatedAt)
	}

	return &oneShotJobProcessor{
		l:                   l,
		profileStorage:      profileStorage,
		symbolizer:          symbolizer,
		aggregator:          aggregator,
		job:                 job,
		stats:               stats,
		degreeOfParallelism: degreeOfParallelism,
		profilesBatchSize:   profilesBatchSize,
		startedAt:           time.Now(),
	}
}

type oneShotJobResult struct {
	profilesProcessed int
	executionStats    JobExecutionStats
}

func (p *oneShotJobProcessor) run(ctx context.Context) (result oneShotJobResult, err error) {
	defer func() {
		stats := *p.stats
		stats.Duration = time.Since(p.startedAt)
		if err != nil {
			stats.Error = err.Error()
		}
		result.executionStats = stats
	}()

	selector, err := buildSelector(p.job.Service, p.job.TimeRange, p.job.PodID, p.job.NodeID)
	if err != nil {
		return oneShotJobResult{}, err
	}

	metadataStart := time.Now()
	profileMetas, err := p.profileStorage.SelectProfiles(ctx, &meta.ProfileQuery{
		Selector: selector,
	})
	p.stats.Stages.QueryMetadata = time.Since(metadataStart)
	if err != nil {
		return oneShotJobResult{}, err
	}

	p.stats.Metrics.ProfilesDiscovered = len(profileMetas)
	if len(profileMetas) == 0 {
		p.l.Warn(ctx, "Job has no profiles, marking as done",
			log.String("service", p.job.Service),
			log.String("workload_key", p.job.WorkloadKey()),
		)
		return oneShotJobResult{}, nil
	}

	p.l.Info(ctx, "Starting job processing",
		log.String("service", p.job.Service),
		log.String("workload_key", p.job.WorkloadKey()),
		log.String("pod_id", p.job.PodID),
		log.String("node_id", p.job.NodeID),
		log.Int("profilesCount", len(profileMetas)),
	)

	buildIDs := getBuildIDsFromProfiles(profileMetas)
	p.stats.Metrics.BuildIDs = len(buildIDs)
	p.stats.Metrics.Batches = (len(profileMetas) + p.profilesBatchSize - 1) / p.profilesBatchSize

	processed, err := p.processProfiles(ctx, profileMetas, buildIDs)
	processed.stages.applyTo(&p.stats.Stages)
	p.stats.ProcessingWall = processed.processingWall
	if err != nil {
		p.stats.Metrics.ProfilesProcessed = len(profileMetas)
		return oneShotJobResult{profilesProcessed: len(profileMetas)}, err
	}

	p.stats.Metrics.Functions = len(processed.functions)
	p.stats.Metrics.ProfilesProcessed = len(profileMetas)

	saveStart := time.Now()
	err = p.aggregator.Save(ctx, &ServicePerfTop{
		Generation:  p.job.Generation,
		ServiceName: p.job.Service,
		Functions:   processed.functions,
	})
	p.stats.Stages.SaveTop = time.Since(saveStart)
	return oneShotJobResult{profilesProcessed: len(profileMetas)}, err
}

type parallelProcessingStages struct {
	DownloadGsym      time.Duration
	FetchProfile      time.Duration
	ParseProfile      time.Duration
	AggregateProfiles time.Duration
	MergeExtract      time.Duration
}

func (s parallelProcessingStages) applyTo(stages *JobStages) {
	stages.DownloadGsym = s.DownloadGsym
	stages.FetchProfile = s.FetchProfile
	stages.ParseProfile = s.ParseProfile
	stages.AggregateProfiles = s.AggregateProfiles
	stages.MergeExtract = s.MergeExtract
}

type processedProfilesResult struct {
	functions      []Function
	stages         parallelProcessingStages
	processingWall time.Duration
}

func (p *oneShotJobProcessor) processProfiles(
	ctx context.Context,
	profileMetas []*meta.ProfileMetadata,
	buildIDs []string,
) (result processedProfilesResult, err error) {
	metaBatchesChan := make(
		chan []*meta.ProfileMetadata,
		(len(profileMetas)+p.profilesBatchSize-1)/p.profilesBatchSize,
	)
	for i := 0; i < len(profileMetas); i += p.profilesBatchSize {
		metaBatchesChan <- profileMetas[i:min(i+p.profilesBatchSize, len(profileMetas))]
	}
	close(metaBatchesChan)

	gsymStart := time.Now()
	gsyms, err := p.symbolizer.DownloadAllGSYMs(ctx, buildIDs)
	result.stages.DownloadGsym = time.Since(gsymStart)
	if err != nil {
		return result, err
	}
	defer gsyms.Release()

	aggregators := make([]*PerfTopAggregator, p.degreeOfParallelism)
	defer func() {
		for _, aggregator := range aggregators {
			if aggregator != nil {
				aggregator.Destroy()
			}
		}
	}()

	processedProfiles := atomic.Int64{}
	profileFetchNs := atomic.Int64{}
	profileParseNs := atomic.Int64{}
	aggregateNs := atomic.Int64{}

	processingStart := time.Now()
	g, ctx := errgroup.WithContext(ctx)
	for i := range p.degreeOfParallelism {
		g.Go(func() error {
			aggregator, err := p.symbolizer.NewPerfTopAggregator()
			if err != nil {
				return err
			}
			aggregators[i] = aggregator

			aggregator.InitializeSymbolizersWithGSYMs(gsyms, buildIDs)

			for metaBatch := range metaBatchesChan {
				batch, err := p.fetchProfiles(ctx, metaBatch)
				if err != nil {
					return err
				}
				profileFetchNs.Add(batch.fetchDuration.Nanoseconds())
				profileParseNs.Add(batch.parseDuration.Nanoseconds())

				p.l.Info(
					ctx,
					"Got a batch of profiles to process",
					log.String("service", p.job.Service),
					log.Int("batchSize", len(batch.profiles)),
					log.Int("alreadyProcessedPct", int(processedProfiles.Load()*100/int64(len(profileMetas)))),
				)

				aggregateStart := time.Now()
				err = aggregator.AddProfiles(ctx, batch.profiles)
				aggregateNs.Add(time.Since(aggregateStart).Nanoseconds())
				if err != nil {
					return err
				}
				processedProfiles.Add(int64(len(batch.profiles)))
			}

			return nil
		})
	}
	if err := g.Wait(); err != nil {
		result.processingWall = time.Since(processingStart)
		result.stages.FetchProfile = time.Duration(profileFetchNs.Load())
		result.stages.ParseProfile = time.Duration(profileParseNs.Load())
		result.stages.AggregateProfiles = time.Duration(aggregateNs.Load())
		return result, err
	}
	result.processingWall = time.Since(processingStart)
	result.stages.FetchProfile = time.Duration(profileFetchNs.Load())
	result.stages.ParseProfile = time.Duration(profileParseNs.Load())
	result.stages.AggregateProfiles = time.Duration(aggregateNs.Load())

	mergeStart := time.Now()
	for i := 1; i < len(aggregators); i += 1 {
		aggregators[0].MergeWith(aggregators[i])
	}
	result.functions = aggregators[0].Extract()
	result.stages.MergeExtract = time.Since(mergeStart)
	return result, nil
}

type fetchedProfilesBatch struct {
	profiles      []profile.ProfileData
	fetchDuration time.Duration
	parseDuration time.Duration
}

func (p *oneShotJobProcessor) fetchProfiles(
	ctx context.Context,
	profileMetas []*meta.ProfileMetadata,
) (result fetchedProfilesBatch, err error) {
	profiles := make([]profile.ProfileData, len(profileMetas))
	var fetchNs atomic.Int64
	var parseNs atomic.Int64

	g, ctx := errgroup.WithContext(ctx)
	for i := range profileMetas {
		g.Go(func() error {
			fetchStart := time.Now()
			profileBundle, err := p.profileStorage.FetchProfile(ctx, profileMetas[i])
			fetchNs.Add(time.Since(fetchStart).Nanoseconds())
			if err != nil {
				noExistErr := &blob.ErrNoExist{}
				if errors.As(err, &noExistErr) {
					return nil
				}
				return err
			}

			parseStart := time.Now()
			pprofData, err := profileBundle.GetOrConvertPprof()
			parseNs.Add(time.Since(parseStart).Nanoseconds())
			if err != nil {
				return err
			}
			profiles[i] = pprofData

			return nil
		})
	}

	err = g.Wait()
	if err != nil {
		result.fetchDuration = time.Duration(fetchNs.Load())
		result.parseDuration = time.Duration(parseNs.Load())
		return result, err
	}

	result.profiles = profiles
	result.fetchDuration = time.Duration(fetchNs.Load())
	result.parseDuration = time.Duration(parseNs.Load())
	return result, nil
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
