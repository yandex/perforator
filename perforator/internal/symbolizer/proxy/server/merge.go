package server

import (
	"context"
	"fmt"
	_ "net/http/pprof"
	"time"

	pprof "github.com/google/pprof/profile"
	"go.opentelemetry.io/otel"
	otelcodes "go.opentelemetry.io/otel/codes"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/yandex/perforator/library/go/ptr"
	"github.com/yandex/perforator/observability/lib/querylang"
	"github.com/yandex/perforator/perforator/internal/symbolizer/symbolize"
	"github.com/yandex/perforator/perforator/pkg/profile/merge"
	"github.com/yandex/perforator/perforator/pkg/profile/quality"
	"github.com/yandex/perforator/perforator/pkg/profile/samplefilter"
	profilestorage "github.com/yandex/perforator/perforator/pkg/storage/profile"
	"github.com/yandex/perforator/perforator/pkg/storage/profile/meta"
	"github.com/yandex/perforator/perforator/proto/perforator"
	profileproto "github.com/yandex/perforator/perforator/proto/profile"
)

// MergeProfiles implements perforator.PerforatorServer
func (s *PerforatorServer) MergeProfiles(
	ctx context.Context,
	req *perforator.MergeProfilesRequest,
) (res *perforator.MergeProfilesResponse, err error) {
	ctx, span := otel.Tracer("APIProxy").Start(ctx, "PerforatorServer.MergeProfiles")
	defer span.End()
	defer func() {
		if err != nil {
			span.SetStatus(otelcodes.Error, err.Error())
			span.RecordError(err)
			s.metrics.mergeProfilesRequests.fails.Inc()
		} else {
			s.metrics.mergeProfilesRequests.successes.Inc()
		}
	}()

	query, err := s.parseProfileQuery(req.GetQuery())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid profile query: %v", err)
	}

	if query.MaxSamples == 0 {
		query.MaxSamples = uint64(req.MaxSamples)
	}

	targetEventType, err := deriveEventTypeFromSelector(query.Selector)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid event type matcher in selector: %v", err)
	}

	metas, err := s.selectProfiles(ctx, query)

	if len(metas) == 0 {
		return nil, status.Errorf(codes.NotFound, "no profiles found")
	}

	var samplePeriod uint64 = 0
	if req.GetExperimental().GetSampleProfileStacks() {
		const kDefaultSamplingTarget = 23
		samplePeriod = calculateSamplePeriod(len(metas), kDefaultSamplingTarget)
	}

	var profile *pprof.Profile
	if req.GetExperimental().GetEnableNewProfileMerger() || (s.c.FeaturesConfig.EnableNewProfileMerger != nil && *s.c.FeaturesConfig.EnableNewProfileMerger) {
		s.l.Debug(ctx, "Merging profiles via new profile merger")
		opts, err := fillMergeOptions(req.MergeOptions, query.Selector, targetEventType, samplePeriod)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to fill merge options: %v", err)
		}
		profile, metas, err = s.fetchAndMergeProfilesFast(ctx, metas, query.Selector, opts)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to merge profiles: %v", err)
		}
	} else {
		s.l.Debug(ctx, "Merging profiles via legacy profile merger")
		profile, metas, err = s.fetchAndMergeProfilesLegacy(ctx, metas, query.Selector, targetEventType, samplePeriod)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to merge profiles: %v", err)
		}
	}

	renderedProfile, err := s.renderProfile(ctx, profile, req.GetFormat())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to render profile: %v", err)
	}

	statistics := quality.CalculateProfileStatistics(profile)

	protometas := make([]*perforator.ProfileMeta, len(metas))
	for i, meta := range metas {
		protometas[i] = storageMetaToProtoMeta(meta)
	}

	res, err = s.makeMergeResponse(ctx, renderedProfile, protometas, statistics, req)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to make merge response: %v", err)
	}
	return res, nil
}

func (s *PerforatorServer) fetchAndMergeProfilesLegacy(
	ctx context.Context,
	metas []*meta.ProfileMetadata,
	selector *querylang.Selector,
	targetEventType string,
	samplePeriod uint64,
) (*pprof.Profile, []*meta.ProfileMetadata, error) {
	const kDefaultDownloadConcurrency = 256

	var datas []profilestorage.ProfileData
	var err error

	if samplePeriod != 0 {
		metas, datas, err = s.sampleProfiles(ctx, metas, targetEventType, samplePeriod)
		if err != nil {
			return nil, nil, err
		}
	} else {
		datas, err = s.fetchProfiles(ctx, metas, kDefaultDownloadConcurrency, nil)
		if err != nil {
			return nil, nil, err
		}
	}

	profiles, err := s.parseProfiles(ctx, datas)
	if err != nil {
		return nil, nil, err
	}

	filters, err := samplefilter.ExtractSelectorFilters(selector)
	if err != nil {
		return nil, nil, err
	}
	postprocessedProfiles := samplefilter.FilterProfilesBySampleFilters(profiles, filters...)

	for _, profile := range postprocessedProfiles {
		fixupMultiSampleTypeProfile(profile, targetEventType)
	}

	mergedProfile, err := s.mergeProfiles(ctx, postprocessedProfiles)
	if err != nil {
		return nil, nil, err
	}

	return mergedProfile, metas, nil
}

func (s *PerforatorServer) fetchAndMergeProfilesFast(
	ctx context.Context,
	metas []*meta.ProfileMetadata,
	selector *querylang.Selector,
	mergeOpts *profileproto.MergeOptions,
) (*pprof.Profile, []*meta.ProfileMetadata, error) {
	const kDefaultDownloadConcurrency = 256

	session, err := s.mergemanager.Start(mergeOpts)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to initialize profile merge session: %w", err)
	}
	defer session.Close()

	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(kDefaultDownloadConcurrency)

	for _, meta := range metas {
		g.Go(func() error {
			profileBundle, err := s.profileStorage.FetchProfile(ctx, meta)
			if err != nil {
				return err
			}
			data, err := profileBundle.GetOrConvertPprof()
			if err != nil {
				return err
			}
			// TODO(ayles) improve stability.
			return session.AddPProfProfile(data)
		})
	}

	if err := g.Wait(); err != nil {
		return nil, nil, err
	}

	cprofile, err := session.Finish()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to merge profiles: %w", err)
	}
	defer cprofile.Free()

	data, err := cprofile.MarshalPProf()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to merge pprof: %w", err)
	}

	mergedProfile, err := pprof.ParseData(data)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to serialize pprof: %w", err)
	}

	return mergedProfile, metas, nil
}

func fillMergeOptions(
	opts *profileproto.MergeOptions,
	selector *querylang.Selector,
	targetEventType string,
	samplePeriod uint64,
) (*profileproto.MergeOptions, error) {
	if opts == nil {
		opts = &profileproto.MergeOptions{}
	}

	if opts.LabelFilter == nil {
		opts.LabelFilter = &profileproto.LabelFilter{}
	}

	if opts.SampleFilter == nil {
		opts.SampleFilter = &profileproto.SampleFilter{}
	}

	if opts.ValueTypeFilter == nil {
		opts.ValueTypeFilter = &profileproto.ValueTypeFilter{}
	}

	getLabelKey := func(label profileproto.WellKnownLabel) string {
		return proto.GetExtension(label.Descriptor().Values().ByNumber(protoreflect.EnumNumber(label)).Options(), profileproto.E_LabelKey).(string)
	}

	opts.LabelFilter.KeysShow = append(opts.LabelFilter.KeysShow, []string{
		getLabelKey(profileproto.WellKnownLabel_ProcessCommand),
		getLabelKey(profileproto.WellKnownLabel_ThreadCommand),
		getLabelKey(profileproto.WellKnownLabel_Workload),
		getLabelKey(profileproto.WellKnownLabel_SignalName),
	}...)

	opts.ValueTypeFilter.Allowlist = append(opts.ValueTypeFilter.Allowlist, []string{targetEventType}...)

	err := samplefilter.FillProtoSampleFilter(selector, opts.SampleFilter)
	if err != nil {
		return nil, err
	}

	if samplePeriod != 0 && opts.SamplePeriod == nil {
		opts.SamplePeriod = ptr.Uint64(samplePeriod)
	}

	return opts, nil
}

func (s *PerforatorServer) makeMergeResponse(
	ctx context.Context,
	profile []byte,
	meta []*perforator.ProfileMeta,
	statistics *perforator.ProfileStatistics,
	req *perforator.MergeProfilesRequest,
) (*perforator.MergeProfilesResponse, error) {
	url, err := s.maybeUploadProfile(ctx, profile, req.GetFormat())
	if err != nil {
		return nil, err
	}

	if url != "" {
		return &perforator.MergeProfilesResponse{
			Result:      &perforator.MergeProfilesResponse_ProfileURL{ProfileURL: url},
			ProfileMeta: meta,
			Statistics:  statistics,
		}, nil
	} else {
		return &perforator.MergeProfilesResponse{
			Result:      &perforator.MergeProfilesResponse_Profile{Profile: profile},
			ProfileMeta: meta,
			Statistics:  statistics,
		}, nil
	}
}

func calculateSamplePeriod(profilesCount int, samplingTarget int) uint64 {
	if samplingTarget == 0 {
		return 0
	}

	samplePeriod := uint64(profilesCount) / uint64(samplingTarget)
	if samplePeriod <= 2 {
		// No point in performing sampling, can just merge as is.
		return 0
	}

	if samplePeriod%2 == 0 {
		// Make samplePeriod odd to avoid potential biases:
		// we collect stacks from every core, there's usually an even number of cores,
		// theoretically with an even ratio we could end up only processing stacks
		// from cores with an even index.
		samplePeriod += 1
	}

	return samplePeriod
}

func (s *PerforatorServer) sampleProfiles(
	ctx context.Context,
	metas []*meta.ProfileMetadata,
	targetEventType string,
	samplePeriod uint64,
) (sampledProfileMetas []*meta.ProfileMetadata, sampledProfileDatas []profilestorage.ProfileData, err error) {
	ctx, span := otel.Tracer("APIProxy").Start(ctx, "PerforatorServer.sampleProfiles")
	defer span.End()
	defer func() {
		if err != nil {
			span.SetStatus(otelcodes.Error, err.Error())
			span.RecordError(err)
		}
	}()

	const kConcurrencyLevel = 16
	const kBatchSize = 20

	sampledProfileDatas = make([]profilestorage.ProfileData, kConcurrencyLevel)
	sampledProfileMetas = make([]*meta.ProfileMetadata, kConcurrencyLevel)

	metasBatches := make(
		chan []*meta.ProfileMetadata,
		(len(metas)+kBatchSize-1)/kBatchSize,
	)
	for i := 0; i < len(metas); i += kBatchSize {
		metasBatches <- metas[i:min(i+kBatchSize, len(metas))]
	}
	close(metasBatches)

	g, ctx := errgroup.WithContext(ctx)
	for i := range kConcurrencyLevel {
		g.Go(func() error {
			sampler, err := symbolize.NewStacksSampler(targetEventType, samplePeriod)
			if err != nil {
				return err
			}
			defer sampler.Destroy()

			for metasBatch := range metasBatches {
				datas, err := s.fetchProfiles(ctx, metasBatch, kBatchSize, nil)
				if err != nil {
					return err
				}

				for j, meta := range metasBatch {
					sampler.AddProfile(datas[j])
					// GC the thing
					datas[j] = nil

					// This is nonsense, obviously,
					// but idk what should actually be in meta when we sample this way
					sampledProfileMetas[i] = meta
				}
			}

			sampledProfileDatas[i], err = sampler.ExtractSampledProfile()
			if err != nil {
				return err
			}

			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, nil, err
	}

	insertIndex := 0
	for i := range sampledProfileDatas {
		if len(sampledProfileDatas[i]) == 0 {
			continue
		}

		sampledProfileDatas[insertIndex] = sampledProfileDatas[i]
		sampledProfileMetas[insertIndex] = sampledProfileMetas[i]
		insertIndex += 1
	}

	return sampledProfileMetas[:insertIndex], sampledProfileDatas[:insertIndex], nil
}

func (s *PerforatorServer) parseProfile(ctx context.Context, data profilestorage.ProfileData) (profile *pprof.Profile, err error) {
	_, span := otel.Tracer("APIProxy").Start(ctx, "PerforatorServer.parseProfile")
	defer span.End()
	defer func() {
		if err != nil {
			span.SetStatus(otelcodes.Error, err.Error())
			span.RecordError(err)
		}
	}()

	profile, err = pprof.ParseData(data)
	return
}

func (s *PerforatorServer) parseProfiles(
	ctx context.Context,
	datas []profilestorage.ProfileData,
) (profiles []*pprof.Profile, err error) {
	ctx, span := otel.Tracer("APIProxy").Start(ctx, "PerforatorServer.parseProfiles")
	defer span.End()
	defer func() {
		if err != nil {
			span.SetStatus(otelcodes.Error, err.Error())
			span.RecordError(err)
		}
	}()

	profiles = make([]*pprof.Profile, len(datas))

	g, ctx := errgroup.WithContext(ctx)

	for i, data := range datas {
		g.Go(func() error {
			var errParse error
			profiles[i], errParse = s.parseProfile(ctx, data)
			return errParse
		})
	}

	err = g.Wait()
	if err != nil {
		return
	}

	return
}

func (s *PerforatorServer) mergeProfiles(
	ctx context.Context,
	profiles []*pprof.Profile,
) (
	res *pprof.Profile,
	err error,
) {
	start := time.Now()
	_, span := otel.Tracer("APIProxy").Start(ctx, "PerforatorServer.mergeProfiles")
	defer span.End()
	defer func() {
		if err != nil {
			span.SetStatus(otelcodes.Error, err.Error())
			span.RecordError(err)
		} else {
			s.metrics.mergeProfilesTimer.RecordDuration(time.Since(start))
		}
	}()

	g, _ := errgroup.WithContext(ctx)
	for _, profile := range profiles {
		profile := profile
		g.Go(func() error {
			if err := cleanupTransientLabels(profile); err != nil {
				return err
			}
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}

	return merge.Merge(profiles)
}
