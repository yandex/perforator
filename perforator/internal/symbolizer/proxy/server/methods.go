package server

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"runtime/debug"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gofrs/uuid"
	pprof "github.com/google/pprof/profile"
	"go.opentelemetry.io/otel"
	otelcodes "go.opentelemetry.io/otel/codes"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/observability/lib/querylang"
	"github.com/yandex/perforator/observability/lib/querylang/operator"
	"github.com/yandex/perforator/perforator/internal/symbolizer/autofdo"
	"github.com/yandex/perforator/perforator/internal/symbolizer/symbolize"
	"github.com/yandex/perforator/perforator/pkg/profile/flamegraph/render"
	"github.com/yandex/perforator/perforator/pkg/profile/interpreterstack"
	"github.com/yandex/perforator/perforator/pkg/profilequerylang"
	"github.com/yandex/perforator/perforator/pkg/sampletype"
	blob "github.com/yandex/perforator/perforator/pkg/storage/blob/models"
	profilestorage "github.com/yandex/perforator/perforator/pkg/storage/profile"
	"github.com/yandex/perforator/perforator/pkg/storage/profile/meta"
	"github.com/yandex/perforator/perforator/pkg/storage/util"
	"github.com/yandex/perforator/perforator/pkg/xlog"
	"github.com/yandex/perforator/perforator/proto/lib/time_interval"
	"github.com/yandex/perforator/perforator/proto/perforator"
	symbolizerproto "github.com/yandex/perforator/perforator/proto/symbolizer"
)

var (
	ErrFailedGetProfile = errors.New("failed to get profile")
)

// TODO(itrofimow): make this 8 configurable?
const PGODegreeOfParallelism uint64 = 8

// ListServices implements perforator.PerforatorServer
func (s *PerforatorServer) ListServices(
	ctx context.Context,
	req *perforator.ListServicesRequest,
) (*perforator.ListServicesResponse, error) {
	var err error
	defer func() {
		if err != nil {
			s.metrics.listServicesRequest.fails.Inc()
		} else {
			s.metrics.listServicesRequest.successes.Inc()
		}
	}()

	if req.Prefix != "" {
		err = status.Errorf(codes.Unimplemented, "prefix is not supported yet")
		return nil, err
	}

	var pruneInterval time.Duration
	if req.MaxStaleAge == nil {
		pruneInterval = s.c.ListServicesSettings.DefaultMaxStaleAge
	} else {
		pruneInterval = req.MaxStaleAge.AsDuration()
	}

	var services []*meta.ServiceMetadata
	services, err = s.profileStorage.ListServices(
		ctx,
		&meta.ServiceQuery{
			Pagination: util.Pagination{
				Offset: uint64(req.Paginated.Offset),
				Limit:  uint64(req.Paginated.Limit),
			},
			SortOrder:   util.SortOrderFromServicesProto(req.OrderBy),
			Regex:       req.Regex,
			MaxStaleAge: &pruneInterval,
		},
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list services: %v", err)
	}

	res := make([]*perforator.ServiceMeta, len(services))
	for i, service := range services {
		res[i] = &perforator.ServiceMeta{
			ServiceID:    service.Service,
			LastUpdate:   timestamppb.New(service.LastUpdate),
			ProfileCount: service.ProfileCount,
		}
	}

	return &perforator.ListServicesResponse{Services: res}, nil
}

// ListSuggestions implements perforator.PerforatorServer
func (s *PerforatorServer) ListSuggestions(
	ctx context.Context,
	req *perforator.ListSuggestionsRequest,
) (*perforator.ListSuggestionsResponse, error) {
	selector := req.GetSelector()
	if selector == "" {
		selector = "{}"
	}
	parsedSelector, err := profilequerylang.ParseSelector(selector)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid selector: %v", err)
	}
	parsedSelector = enrichCPOIDMatcher(parsedSelector)

	parsedSelector = replaceSelectorTimeInterval(parsedSelector, &time_interval.TimeInterval{
		From: timestamppb.New(time.Now().Add(-time.Hour * 24 * 14)),
		To:   timestamppb.Now(),
	})
	query := &meta.SuggestionsQuery{
		Field:    req.Field,
		Regex:    req.Regex,
		Selector: parsedSelector,
	}
	if req.Paginated != nil {
		query.Pagination = util.Pagination{
			Offset: uint64(req.Paginated.Offset),
			Limit:  uint64(req.Paginated.Limit),
		}
	}

	suggestions, err := s.profileStorage.ListSuggestions(
		ctx,
		query,
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list suggestions: %v", err)
	}

	if suggestions == nil {
		return &perforator.ListSuggestionsResponse{
			SuggestSupported: false,
		}, nil
	}

	res := make([]*perforator.Suggestion, len(suggestions))
	for i, suggestion := range suggestions {
		res[i] = &perforator.Suggestion{
			Value: suggestion.Value,
		}
	}

	return &perforator.ListSuggestionsResponse{
		SuggestSupported: true,
		Suggestions:      res,
	}, nil
}

func storageMetaToProtoMeta(meta *meta.ProfileMetadata) *perforator.ProfileMeta {
	return &perforator.ProfileMeta{
		ProfileID:  meta.ID,
		System:     meta.System,
		EventType:  meta.MainEventType,
		Cluster:    meta.Cluster,
		Service:    meta.Service,
		PodID:      meta.PodID,
		NodeID:     meta.NodeID,
		Timestamp:  timestamppb.New(meta.Timestamp),
		BuildIDs:   slices.Clone(meta.BuildIDs),
		Attributes: maps.Clone(meta.Attributes),
		CPOID:      meta.CustomProfilingOperationID,
	}
}

func parseProfileSelector(query *perforator.ProfileQuery) (*querylang.Selector, error) {
	if query.Selector == "" {
		return nil, errors.New("selector is required")
	}

	selector, err := profilequerylang.ParseSelector(query.Selector)
	if err != nil {
		return nil, err
	}

	if ts := query.GetTimeInterval().GetFrom(); ts != nil {
		selector.Matchers = append(
			selector.Matchers,
			profilequerylang.BuildMatcher(
				profilequerylang.TimestampLabel,
				querylang.AND,
				querylang.Condition{Operator: operator.GTE},
				[]string{ts.AsTime().Format(time.RFC3339Nano)},
			),
		)
	}
	if ts := query.GetTimeInterval().GetTo(); ts != nil {
		selector.Matchers = append(
			selector.Matchers,
			profilequerylang.BuildMatcher(
				profilequerylang.TimestampLabel,
				querylang.AND,
				querylang.Condition{Operator: operator.LTE},
				[]string{ts.AsTime().Format(time.RFC3339Nano)},
			),
		)
	}

	return selector, nil
}

func isRawRenderFormat(format *perforator.RenderFormat) bool {
	_, isRawFormat := format.GetFormat().(*perforator.RenderFormat_RawProfile)

	return isRawFormat
}

func (s *PerforatorServer) buildExcludeProfilerVersionMatcher() *querylang.Matcher {
	return profilequerylang.BuildMatcher(
		profilequerylang.ProfilerVersionLabel,
		querylang.AND,
		querylang.Condition{
			Operator: operator.Eq,
			Inverse:  true,
		},
		s.c.ProfileBlacklist.ProfilerVersions,
	)
}

func enrichCPOIDMatcher(selector *querylang.Selector) *querylang.Selector {
	// if cpo_id matcher is already present, return the selector
	for _, matcher := range selector.Matchers {
		if matcher.Field == profilequerylang.CPOIDLabel {
			return selector
		}

		// If concrete profile is requested via `{id = "xxx"}` we must not add CPOID matcher,
		if matcher.Field == profilequerylang.ProfileIDLabel &&
			len(matcher.Conditions) == 1 &&
			matcher.Conditions[0].Operator == operator.Eq {
			return selector
		}
	}

	// Add matcher for empty cpo_id
	selector.Matchers = append(selector.Matchers, profilequerylang.BuildMatcher(
		profilequerylang.CPOIDLabel,
		querylang.AND,
		querylang.Condition{Operator: operator.Eq},
		[]string{""},
	))

	return selector
}

func (s *PerforatorServer) defaultEventTypeMatcher() *querylang.Matcher {
	return profilequerylang.BuildMatcher(
		profilequerylang.EventTypeLabel,
		querylang.AND,
		querylang.Condition{
			Operator: operator.Eq,
		},
		[]string{sampletype.SampleTypeCPUCycles},
	)
}

func (s *PerforatorServer) parseProfileQuery(query *perforator.ProfileQuery) (*meta.ProfileQuery, error) {
	selector, err := parseProfileSelector(query)
	if err != nil {
		return nil, fmt.Errorf("failed to parse profile selector: %w", err)
	}

	var hasEventTypeMatcher bool
	for _, matcher := range selector.Matchers {
		if matcher.Field == profilequerylang.EventTypeLabel {
			hasEventTypeMatcher = true
			break
		}
	}

	if !hasEventTypeMatcher {
		selector.Matchers = append(selector.Matchers, s.defaultEventTypeMatcher())
	}

	if s.c.ProfileBlacklist != nil && len(s.c.ProfileBlacklist.ProfilerVersions) > 0 {
		selector.Matchers = append(
			selector.Matchers,
			s.buildExcludeProfilerVersionMatcher(),
		)
	}

	selector = enrichCPOIDMatcher(selector)

	return &meta.ProfileQuery{
		Selector:   selector,
		MaxSamples: uint64(query.GetMaxSamples()),
	}, nil
}

// At some point we started to export profiles with multiple SampleTypes, e.g. [{cpu: cycles}, {wall: seconds}],
// however in the database they have a single 'cpu.cycles' EventType.
// This creates problems when we try to merge profiles selected by 'cpu.cycles' EventType:
// both versions might get selected from the DB, but there's no way to merge newer profiles with
// older ones with a single [{cpu: cycles}] SampleType.
//
// This function is a temporary (until wall-time profiles get some love) workaround, which extracts
// a single 'targetEventType' from the profiles and fixes its Samples accordingly.
func fixupMultiSampleTypeProfile(profile *pprof.Profile, targetEventType string) {
	if len(profile.SampleType) <= 1 {
		return
	}

	targetIdx := -1
	for idx, sampleType := range profile.SampleType {
		if sampletype.SampleTypeToString(sampleType) == targetEventType {
			targetIdx = idx
			break
		}
	}
	if targetIdx == -1 {
		return
	}

	for _, sample := range profile.Sample {
		if targetIdx >= len(sample.Value) {
			continue
		}

		sample.Value = []int64{sample.Value[targetIdx]}
	}
	profile.SampleType = []*pprof.ValueType{profile.SampleType[targetIdx]}
}

func deriveEventTypeFromSelector(selector *querylang.Selector) (string, error) {
	equalityConditions := 0
	targetEventType := ""

	for _, matcher := range selector.Matchers {
		if matcher.Field != profilequerylang.EventTypeLabel {
			continue
		}

		for _, condition := range matcher.Conditions {
			if condition.Operator != operator.Eq || condition.Inverse {
				return "", fmt.Errorf("operator %s is unsupported for event_type label", operator.Repr(condition.Operator, condition.Inverse))
			}

			targetEventType = profilequerylang.ValueRepr(condition.Value)
			equalityConditions++
		}
	}

	if equalityConditions != 1 {
		return "", fmt.Errorf("expected one equality condition for event_type, got %d", equalityConditions)
	}

	return targetEventType, nil
}

// ListProfiles implements perforator.PerforatorServer
func (s *PerforatorServer) ListProfiles(
	ctx context.Context,
	req *perforator.ListProfilesRequest,
) (*perforator.ListProfilesResponse, error) {
	var err error
	defer func() {
		if err != nil {
			s.metrics.listProfilesRequests.fails.Inc()
		} else {
			s.metrics.listProfilesRequests.successes.Inc()
		}
	}()

	query, err := s.parseProfileQuery(req.GetQuery())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid profile query: %v", err)
	}

	query.Pagination.Limit = uint64(req.GetPaginated().GetLimit() + 1)
	query.Pagination.Offset = uint64(req.GetPaginated().GetOffset())
	query.SortOrder = util.SortOrderFromProto(req.GetOrderBy())

	var profiles []*meta.ProfileMetadata
	profiles, err = s.profileStorage.SelectProfiles(ctx, query)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to select profiles: %v", err)
	}

	hasMore := len(profiles) == int(query.Pagination.Limit)
	if hasMore {
		profiles = profiles[:len(profiles)-1]
	}

	metas := make([]*perforator.ProfileMeta, 0, len(profiles))
	for _, profile := range profiles {
		metas = append(metas, storageMetaToProtoMeta(profile))
	}

	return &perforator.ListProfilesResponse{
		Profiles: metas,
		HasMore:  hasMore,
	}, nil
}

func makeExactProfileSelector(id string) *querylang.Selector {
	return &querylang.Selector{
		Matchers: []*querylang.Matcher{{
			Field:    profilequerylang.ProfileIDLabel,
			Operator: querylang.OR,
			Conditions: []*querylang.Condition{{
				Operator: operator.Eq,
				Value:    querylang.String{Value: id},
			}},
		}},
	}
}

func (s *PerforatorServer) fetchProfile(ctx context.Context, id meta.ProfileID) (profile *pprof.Profile, m *meta.ProfileMetadata, err error) {
	ctx, span := otel.Tracer("APIProxy").Start(ctx, "PerforatorServer.fetchProfile")
	defer span.End()
	defer func() {
		if err != nil {
			span.SetStatus(otelcodes.Error, err.Error())
			span.RecordError(err)
		}
	}()

	profiles, err := s.profileStorage.SelectProfiles(ctx, &meta.ProfileQuery{
		Selector: makeExactProfileSelector(id),
	})
	if err != nil {
		return
	}

	if len(profiles) == 0 {
		err = ErrFailedGetProfile
		return
	}
	m = profiles[0]

	data, err := s.profileStorage.FetchProfile(ctx, m)
	if err != nil {
		return
	}

	profile, err = pprof.ParseData(data)
	if err != nil {
		return
	}

	return
}

func (s *PerforatorServer) guessBuildIDForPGO(ctx context.Context, datas []profilestorage.ProfileData) (string, error) {
	guesser, err := autofdo.NewBuildIDGuesser(PGODegreeOfParallelism)
	if err != nil {
		return "", err
	}
	defer guesser.Destroy()

	_, span := otel.Tracer("APIProxy").Start(ctx, "PerforatorServer.GuessBuildIDForPGO")
	defer span.End()

	g, _ := errgroup.WithContext(ctx)
	for i := 0; i < int(PGODegreeOfParallelism); i++ {
		guesserIndex := uint64(i)
		g.Go(func() error {
			for j := guesserIndex; j < uint64(len(datas)); j += PGODegreeOfParallelism {
				err := guesser.FeedProfile(guesserIndex, datas[j])
				if err != nil {
					return err
				}
			}
			return nil
		})
	}
	err = g.Wait()

	if err != nil {
		return "", err
	}

	return guesser.GuessBuildID()
}

func (s *PerforatorServer) processLBRProfiles(
	ctx context.Context,
	metas []*meta.ProfileMetadata,
	datas []profilestorage.ProfileData,
	buildID string,
) (autofdo.ProcessedLBRData, error) {
	builder, err := autofdo.NewBatchInputBuilder(PGODegreeOfParallelism, buildID)
	if err != nil {
		return autofdo.ProcessedLBRData{}, err
	}
	defer builder.Destroy()

	_, span := otel.Tracer("APIProxy").Start(ctx, "PerforatorServer.processLBRProfiles")
	defer span.End()

	g, _ := errgroup.WithContext(ctx)
	for i := 0; i < int(PGODegreeOfParallelism); i++ {
		builderIndex := uint64(i)
		g.Go(func() error {
			for j := builderIndex; j < uint64(len(datas)); j += PGODegreeOfParallelism {
				err := builder.AddProfile(builderIndex, metas[j].Service, datas[j])
				if err != nil {
					return err
				}
			}
			return nil
		})
	}
	err = g.Wait()

	if err != nil {
		return autofdo.ProcessedLBRData{}, err
	}
	return builder.Finalize()
}

type autofdoInput struct {
	Data    autofdo.ProcessedLBRData
	BuildID string
}

// This functions owns a lot of memory in the form of rawProfiles, and we try to GC that
// before running memory-intensive llvm tools, so keep this noinline for a good measure.
//
//go:noinline
func (s *PerforatorServer) generateAutofdoInput(
	ctx context.Context,
	query *meta.ProfileQuery,
	profilesToProcessTotalSizeLimit uint64,
) (autofdoInput, error) {
	metas, err := s.selectProfiles(ctx, query)
	if err != nil {
		return autofdoInput{}, err
	}

	datas, err := s.fetchProfiles(ctx, metas, 256, &profilesToProcessTotalSizeLimit)
	if err != nil {
		return autofdoInput{}, err
	}

	metas, datas = filterNoBlobProfiles(metas, datas)

	buildID, err := s.guessBuildIDForPGO(ctx, datas)
	if err != nil {
		return autofdoInput{}, err
	}

	processesLBR, err := s.processLBRProfiles(ctx, metas, datas, buildID)
	if err != nil {
		return autofdoInput{}, err
	}

	return autofdoInput{
		Data:    processesLBR,
		BuildID: buildID,
	}, nil
}

func (s *PerforatorServer) doGeneratePGOProfile(
	ctx context.Context,
	query *meta.ProfileQuery,
	format *perforator.PGOProfileFormat,
	profilesToProcessTotalSizeLimit uint64,
) ([]byte, *perforator.PGOMeta, error) {
	autofdoInput, err := s.generateAutofdoInput(
		ctx,
		query,
		profilesToProcessTotalSizeLimit,
	)
	if err != nil {
		return nil, nil, err
	}
	// GC the profiles before running memory-intensive llvm tools
	debug.FreeOSMemory()

	autofdoMetadata := autofdoInput.Data.MetaData
	if autofdoMetadata.TotalBranches == 0 {
		return nil, nil, fmt.Errorf("empty autofdo input")
	}

	var profile *LLVMPGOProfile
	switch v := format.GetFormat().(type) {
	case *perforator.PGOProfileFormat_AutoFDO:
		profile, err = s.llvmTools.CreateAutofdoProfile(ctx, []byte(autofdoInput.Data.AutofdoInput), autofdoInput.BuildID)
	case *perforator.PGOProfileFormat_Bolt:
		profile, err = s.llvmTools.CreateBoltProfile(ctx, []byte(autofdoInput.Data.BoltInput), autofdoInput.BuildID)
	default:
		return nil, nil, fmt.Errorf("unsupported PGO render format: %T", v)
	}
	if err != nil {
		return nil, nil, err
	}

	branchesToBytesRatio := float32(0)
	if profile.executableBytesCount != 0 {
		branchesToBytesRatio = float32(float64(autofdoMetadata.TotalBranches) / float64(profile.executableBytesCount))
	}

	return profile.profileBytes, &perforator.PGOMeta{
		TotalProfiles:                       autofdoMetadata.TotalProfiles,
		TotalSamples:                        autofdoMetadata.TotalSamples,
		TotalBranches:                       autofdoMetadata.TotalBranches,
		BogusLbrEntries:                     autofdoMetadata.BogusLbrEntries,
		TakenBranchesToExecutableBytesRatio: branchesToBytesRatio,
		BranchCountMapSize:                  autofdoMetadata.BranchCountMapSize,
		RangeCountMapSize:                   autofdoMetadata.RangeCountMapSize,
		AddressCountMapSize:                 autofdoMetadata.AddressCountMapSize,
		GuessedBuildID:                      autofdoInput.BuildID,
		ProfilesByServiceCount:              autofdoMetadata.ProfilesCountByService,
	}, nil
}

// GetProfile implements perforator.PerforatorServer
func (s *PerforatorServer) GetProfile(
	ctx context.Context,
	req *perforator.GetProfileRequest,
) (*perforator.GetProfileResponse, error) {
	var err error
	defer func() {
		if err != nil {
			s.metrics.getProfileRequests.fails.Inc()
		} else {
			s.metrics.getProfileRequests.successes.Inc()
		}
	}()

	l := s.l.With(log.String("method", "GetProfile"))

	l.Info(ctx, "Fetching profile", log.Any("request", req))
	var profile *pprof.Profile
	var meta *meta.ProfileMetadata
	profile, meta, err = s.fetchProfile(ctx, req.GetProfileID())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to fetch profiles: %v", err)
	}

	// TODO: specify target event type in GetProfileRequest
	targetEventType := meta.MainEventType
	if targetEventType != sampletype.SampleTypeLbrStacks {
		targetEventType = sampletype.SampleTypeCPUCycles
	}
	fixupMultiSampleTypeProfile(profile, targetEventType)

	if meta.MainEventType == sampletype.SampleTypeLbrStacks && !isRawRenderFormat(req.GetFormat()) {
		return nil, status.Errorf(codes.Unimplemented, "only RawProfile format is supported for this profile")
	}

	l.Info(ctx, "Rendering profile")
	var buf []byte
	buf, err = s.renderProfile(ctx, profile, req.GetFormat())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to render profile: %v", err)
	}

	return &perforator.GetProfileResponse{
		Profile:     buf,
		ProfileMeta: storageMetaToProtoMeta(meta),
	}, nil
}

func filterNoBlobProfiles(metas []*meta.ProfileMetadata, datas []profilestorage.ProfileData) ([]*meta.ProfileMetadata, []profilestorage.ProfileData) {
	m := make([]*meta.ProfileMetadata, 0, len(metas))
	d := make([]profilestorage.ProfileData, 0, len(datas))
	for i, data := range datas {
		if len(data) > 0 {
			m = append(m, metas[i])
			d = append(d, data)
		}
	}
	return m, d
}

func (s *PerforatorServer) constructPGOProfilesQuery(req *perforator.GeneratePGOProfileRequest) (
	*meta.ProfileQuery,
	error,
) {
	// We deal with large binaries and we want A LOT of lbr-profiles
	// for better resulting sPGO-profile quality. This is basically a
	// "number big enough".
	const maxSamples = uint32(5000)

	// Given that we _guess_ the target buildID, it makes sense to look for
	// somewhat recent profiles only.
	defaultTimeIntervalStart := time.Now().Add(-time.Hour * 24)

	query, err := func() (*meta.ProfileQuery, error) {
		// Old-fashioned request came in, which only contains 'service',
		// nothing else is specified.
		if req.GetQuery() == nil {
			if len(req.GetService()) == 0 {
				return nil, fmt.Errorf("empty 'service' in request")
			}

			selector := fmt.Sprintf("{%s=\"%s\", %s=\"%s\"}",
				profilequerylang.EventTypeLabel, sampletype.SampleTypeLbrStacks,
				profilequerylang.ServiceLabel, req.GetService())

			return s.parseProfileQuery(&perforator.ProfileQuery{
				Selector: selector,
				TimeInterval: &time_interval.TimeInterval{
					From: timestamppb.New(defaultTimeIntervalStart),
				},
			})
		} else {
			// A more modern way to request a PGO-profile, could request profiles from whatever
			// the selector could describe.
			query, err := s.parseProfileQuery(&perforator.ProfileQuery{
				Selector: req.GetQuery().GetSelector(),
			})
			if err != nil {
				return nil, err
			}

			// Now, ensure that selector
			// a) limits profiles to sampletype.SampleTypeLbrStacks event_type
			// b) has a time interval

			// Set the event_type matcher
			query.Selector.Matchers = slices.DeleteFunc(query.Selector.Matchers, func(matcher *querylang.Matcher) bool {
				return matcher.Field == profilequerylang.EventTypeLabel
			})
			query.Selector.Matchers = append(query.Selector.Matchers, profilequerylang.BuildMatcher(
				profilequerylang.EventTypeLabel,
				querylang.AND,
				querylang.Condition{
					Operator: operator.Eq,
				},
				[]string{sampletype.SampleTypeLbrStacks},
			))
			// Ensure TimeInterval is specified
			timeInterval, err := profilequerylang.ParseTimeInterval(query.Selector)
			if err != nil {
				return nil, fmt.Errorf("failed to parse selector time interval: %w", err)
			}
			if timeInterval.To == nil && timeInterval.From == nil {
				// Interval is not specified, set it to default values
				query.Selector.Matchers = append(
					query.Selector.Matchers,
					profilequerylang.BuildMatcher(
						profilequerylang.TimestampLabel,
						querylang.AND,
						querylang.Condition{Operator: operator.GTE},
						[]string{defaultTimeIntervalStart.Format(time.RFC3339Nano)},
					),
				)
			} else if timeInterval.To == nil || timeInterval.From == nil {
				return nil, fmt.Errorf(
					"failed to parse selector: either none or both time interval bounds should be specified",
				)
			}

			return query, nil
		}
	}()
	if err != nil {
		return nil, err
	}

	// This increases the chance of guessing a buildID from the most recent release,
	// instead of the previous one(s).
	query.SortOrder = util.SortOrder{
		Columns: []util.SortColumn{
			{
				Name:       profilequerylang.TimestampLabel,
				Descending: true,
			},
		},
	}
	query.MaxSamples = 0
	query.Pagination = util.Pagination{
		Offset: 0,
		Limit:  uint64(maxSamples),
	}

	return query, nil
}

func (s *PerforatorServer) GeneratePGOProfile(
	ctx context.Context,
	req *perforator.GeneratePGOProfileRequest,
) (*perforator.GeneratePGOProfileResponse, error) {
	query, err := s.constructPGOProfilesQuery(req)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid PGO profiles query: %v", err)
	}

	// Some services are loaded heavier than others, and thus generate larger profiles.
	// We don't want to run out of memory when downloading 5k of huge profiles
	// (10Mb per-profile is not uncommon, which would amount to whopping 50GB of data).
	// Presumably, 8Gb of data should be more than enough, and we would either hit
	// "maxSamples" limit for not-that-heavy-loaded services, or this one.
	const profilesToProcessTotalSizeLimit = uint64(8 * 1024 * 1024 * 1024)
	buf, PGOmeta, err := s.doGeneratePGOProfile(ctx, query, req.GetFormat(), profilesToProcessTotalSizeLimit)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to generate PGO profile: %v", err)
	}

	id, err := s.makePGOProfileKey(req.GetFormat())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to make PGO profile key: %v", err)
	}

	url, err := s.maybeUploadProfileWithID(ctx, buf, id)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to upload PGO profile: %v", err)
	}

	if url != "" {
		return &perforator.GeneratePGOProfileResponse{
			Result:  &perforator.GeneratePGOProfileResponse_ProfileURL{ProfileURL: url},
			PGOMeta: PGOmeta,
		}, nil
	} else {
		return &perforator.GeneratePGOProfileResponse{
			Result:  &perforator.GeneratePGOProfileResponse_Profile{Profile: buf},
			PGOMeta: PGOmeta,
		}, nil
	}
}

func (s *PerforatorServer) selectProfiles(
	ctx context.Context,
	query *meta.ProfileQuery,
) (metas []*meta.ProfileMetadata, err error) {
	ctx, span := otel.Tracer("APIProxy").Start(ctx, "PerforatorServer.selectProfiles")
	defer span.End()
	defer func() {
		if err != nil {
			span.SetStatus(otelcodes.Error, err.Error())
			span.RecordError(err)
		}
	}()

	const kDefaultMaxSamples = 10

	if query.MaxSamples == 0 && query.Limit == 0 {
		query.MaxSamples = kDefaultMaxSamples
	}

	return s.profileStorage.SelectProfiles(ctx, query)
}

// If @totalSizeSoftLimit is set, stop downloading profiles when the volume of
// downloaded profiles exceeds @totalSizeSoftLimit.
func (s *PerforatorServer) fetchProfiles(
	ctx context.Context,
	metas []*meta.ProfileMetadata,
	downloadConcurrency int,
	totalSizeSoftLimit *uint64,
) (datas []profilestorage.ProfileData, err error) {
	ctx, span := otel.Tracer("APIProxy").Start(ctx, "PerforatorServer.fetchProfiles")
	defer span.End()
	defer func() {
		if err != nil {
			span.SetStatus(otelcodes.Error, err.Error())
			span.RecordError(err)
		}
	}()

	var downloadedSizeApprox atomic.Uint64
	var droppedProfilesCount atomic.Uint64

	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(downloadConcurrency)

	datas = make([]profilestorage.ProfileData, len(metas))

	for i, meta := range metas {
		g.Go(func() error {
			if totalSizeSoftLimit != nil && downloadedSizeApprox.Load() >= *totalSizeSoftLimit {
				droppedProfilesCount.Add(1)
				return nil
			}

			data, err := s.profileStorage.FetchProfile(ctx, meta)
			noExistErr := &blob.ErrNoExist{}
			if err != nil && !errors.As(err, &noExistErr) {
				return err
			}

			downloadedSizeApprox.Add(uint64(len(data)))
			datas[i] = data

			return nil
		})
	}

	err = g.Wait()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to render profile: %v", err)
	}

	droppedProfiles := droppedProfilesCount.Load()
	if droppedProfiles != 0 {
		s.l.Warn(
			ctx,
			"Some profiles were not loaded due to memory limits",
			log.UInt64("droppedProfiles", droppedProfiles),
			log.UInt64("downloadedSize", downloadedSizeApprox.Load()),
		)
	}

	return datas, nil
}

func (s *PerforatorServer) DiffProfiles(
	ctx context.Context,
	req *perforator.DiffProfilesRequest,
) (*perforator.DiffProfilesResponse, error) {
	var err error
	defer func() {
		if err != nil {
			s.metrics.diffProfilesRequests.fails.Inc()
		} else {
			s.metrics.diffProfilesRequests.successes.Inc()
		}
	}()

	fgm := sync.Mutex{}
	fg := render.NewFlameGraph()
	var fgOptions *perforator.FlamegraphOptions
	var fgFormat render.Format
	if req.RenderFormat == nil {
		fgOptions = req.GetFlamegraphOptions()
		fgFormat = render.HTMLFormat
	} else {
		switch v := (req.RenderFormat.Format).(type) {
		case *perforator.RenderFormat_Flamegraph:
			fgOptions = v.Flamegraph
			fgFormat = render.HTMLFormat
		case *perforator.RenderFormat_HTMLVisualisation:
			fgOptions = v.HTMLVisualisation
			fgFormat = render.HTMLFormat
		case *perforator.RenderFormat_JSONFlamegraph:
			fgOptions = v.JSONFlamegraph
			fgFormat = render.JSONFormat
		default:
			return nil, status.Errorf(codes.Unimplemented, "unsupported diff render format %T", v)
		}
	}

	err = fillFlamegraphOptions(fg, fgOptions)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid flamegraph options: %v", err)
	}
	fg.SetFormat(fgFormat)

	taskIDs := make([]string, 2)

	populate := func(ctx context.Context, baseline bool) error {
		var query *perforator.ProfileQuery
		if baseline {
			query = req.GetBaselineQuery()
		} else {
			query = req.GetDiffQuery()
		}

		task, err := s.spawnDiffMergeTask(ctx, req, query)
		if err != nil {
			return err
		}

		s.l.Debug(ctx, "Spawned diff subtask", log.String("id", task), log.Bool("baseline", baseline))

		if baseline {
			taskIDs[0] = task
		} else {
			taskIDs[1] = task
		}

		result, err := s.waitTasks(ctx, task)
		if err != nil {
			s.l.Error(ctx, "Diff subtask failed", log.String("id", task), log.Error(err))
			return err
		}

		buf, err := s.downloadArtifact(result[0].GetMergeProfiles().GetProfileURL())
		if err != nil {
			return err
		}

		profile, err := pprof.ParseData(buf)
		if err != nil {
			return err
		}

		{
			fgm.Lock()
			if baseline {
				err = fg.AddBaselineProfile(profile)
			} else {
				err = fg.AddProfile(profile)
			}
			fgm.Unlock()
		}

		if err != nil {
			return err
		}

		return nil
	}

	errg, gctx := errgroup.WithContext(ctx)
	errg.Go(func() error {
		return populate(gctx, true)
	})
	errg.Go(func() error {
		return populate(gctx, false)
	})
	err = errg.Wait()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to process diff profiles: %v", err)
	}

	var buf []byte
	buf, err = fg.RenderBytes()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to render flamegraph: %v", err)
	}

	var renderFormat *perforator.RenderFormat
	if req.GetRenderFormat() != nil {
		renderFormat = req.GetRenderFormat()
	} else {
		renderFormat = &perforator.RenderFormat{
			Symbolize: req.GetSymbolizeOptions(),
			Format: &perforator.RenderFormat_JSONFlamegraph{
				JSONFlamegraph: &perforator.FlamegraphOptions{},
			},
		}
	}
	var url string
	url, err = s.maybeUploadProfile(ctx, buf, renderFormat)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to upload diff profile: %v", err)
	}

	return &perforator.DiffProfilesResponse{
		Result:         &perforator.DiffProfilesResponse_ProfileURL{ProfileURL: url},
		DiffTaskID:     taskIDs[1],
		BaselineTaskID: taskIDs[0],
	}, nil
}

func (s *PerforatorServer) downloadArtifact(url string) ([]byte, error) {
	rsp, err := s.httpclient.R().Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch merged profile: %w", err)
	}
	if !rsp.IsSuccess() {
		return nil, fmt.Errorf("failed to fetch merged profile: got HTTP status %s", rsp.Status())
	}
	return rsp.Body(), nil
}

func (s *PerforatorServer) spawnDiffMergeTask(
	ctx context.Context,
	req *perforator.DiffProfilesRequest,
	query *perforator.ProfileQuery,
) (taskID string, err error) {
	task, err := s.StartTask(ctx, &perforator.StartTaskRequest{
		Spec: &perforator.TaskSpec{
			Kind: &perforator.TaskSpec_MergeProfiles{
				MergeProfiles: &perforator.MergeProfilesRequest{
					Format: &perforator.RenderFormat{
						Symbolize: req.GetSymbolizeOptions(),
						Format: &perforator.RenderFormat_RawProfile{
							RawProfile: &perforator.RawProfileOptions{},
						},
					},
					Query:        query,
					MaxSamples:   query.GetMaxSamples(),
					Experimental: req.GetExperimental(),
				},
			},
		},
	})

	if err != nil {
		return "", err
	}

	return task.TaskID, nil
}

// Stores references to the original profile locations
type pprofBuildIDLocationsOffsetIndex struct {
	locationAtOffset map[uint64]*pprof.Location
}

func prepareRemoteSymbolizeRequest(ctx context.Context, l xlog.Logger, profile *pprof.Profile) ([]*symbolizerproto.PerBinaryRequest, map[string]*pprofBuildIDLocationsOffsetIndex) {
	locsPerBuildID := make(map[string]*pprofBuildIDLocationsOffsetIndex)
	offsetsPerBuildID := make(map[string][]uint64)

	var nilMappingLocs []uint64
	for _, loc := range profile.Location {
		if loc.Mapping == nil {
			nilMappingLocs = append(nilMappingLocs, loc.ID)
			continue
		}
		buildID := loc.Mapping.BuildID
		offset := loc.Address

		locs, ok := locsPerBuildID[buildID]
		if !ok {
			locs = &pprofBuildIDLocationsOffsetIndex{
				locationAtOffset: make(map[uint64]*pprof.Location),
			}
			locsPerBuildID[buildID] = locs
			offsetsPerBuildID[buildID] = make([]uint64, 0)
		}

		offsetsPerBuildID[buildID] = append(offsetsPerBuildID[buildID], offset)
		locs.locationAtOffset[offset] = loc
	}

	if nilMappingLocs != nil {
		l.Warn(ctx, "Couldn't retrieve BuildID: locations with nil mapping found", log.Array("location-id", nilMappingLocs))
	}

	batch := make([]*symbolizerproto.PerBinaryRequest, 0, len(locsPerBuildID))
	for buildID := range locsPerBuildID {
		batch = append(batch, &symbolizerproto.PerBinaryRequest{
			BuildID: buildID,
			Offsets: offsetsPerBuildID[buildID],
		})
	}

	return batch, locsPerBuildID
}

func populatePprofLocation(profile *pprof.Profile, pprofLoc *pprof.Location, loc *symbolizerproto.LocationSymbolizationResult, opts *perforator.SymbolizeOptions) {
	for _, line := range loc.Lines {
		symbolize.AddLine(profile, pprofLoc, line, opts)
	}
}

func (s *PerforatorServer) performRemoteSymbolize(
	ctx context.Context,
	profile *pprof.Profile,
	opts *perforator.SymbolizeOptions,
) (ok bool) {
	batch, locsPerBuildID := prepareRemoteSymbolizeRequest(ctx, s.l, profile)
	resp, failedReqs, err := s.bpClient.DistributeSymbolize(ctx, batch)

	if err != nil {
		// Perform symbolization on proxy instead
		s.l.Error(ctx, "Remote symbolization error: %w", log.Error(err))
		s.metrics.remoteSymbolizationCount.fails.Inc()
		return false
	}

	complete := true
	defer func() {
		s.metrics.remoteSymbolizationCount.successes.Inc()
		if complete {
			s.metrics.remoteSymbolizationCompletenessCount.successes.Inc()
		} else {
			s.metrics.remoteSymbolizationCompletenessCount.fails.Inc()
		}
	}()

	for _, perBinaryResponse := range resp.Batch {
		buildID := perBinaryResponse.BuildID
		if perBinaryResponse.Error != "" {
			complete = false
			s.l.Warn(
				ctx,
				"Remote symbolization for binary failed",
				log.String("buildID", buildID),
				log.String("error", perBinaryResponse.Error),
			)
			continue
		}

		pprofLocs := locsPerBuildID[buildID]
		for _, loc := range perBinaryResponse.Locations {
			if loc.Error != "" {
				complete = false
				s.l.Warn(
					ctx,
					"Remote symbolization for location failed",
					log.String("buildID", buildID),
					log.UInt64("offset", loc.GetELFOffset()),
					log.String("error", loc.Error),
				)
				continue
			}

			pprofLoc := pprofLocs.locationAtOffset[loc.GetELFOffset()]
			populatePprofLocation(profile, pprofLoc, loc, opts)
		}
	}

	return failedReqs == nil
}

func (s *PerforatorServer) symbolizeProfile(
	ctx context.Context,
	profile *pprof.Profile,
	opts *perforator.SymbolizeOptions,
) (res *pprof.Profile, err error) {
	ctx, span := otel.Tracer("APIProxy").Start(ctx, "PerforatorServer.symbolizeProfile")
	defer span.End()
	defer func() {
		if err != nil {
			span.SetStatus(otelcodes.Error, err.Error())
			span.RecordError(err)
		}
	}()

	if opts != nil && opts.Symbolize != nil && !*opts.Symbolize {
		return profile, nil
	}

	if s.bpClient != nil {
		ok := s.performRemoteSymbolize(ctx, profile, opts)

		if ok {
			return profile, nil
		}
	}

	s.metrics.symbolizationProxyFallbackCount.Inc()
	return s.symbolizer.SymbolizeStorageProfile(
		ctx,
		profile,
		opts,
	)
}

func (s *PerforatorServer) renderProfile(
	ctx context.Context,
	profile *pprof.Profile,
	format *perforator.RenderFormat,
) (res []byte, err error) {
	ctx, span := otel.Tracer("APIProxy").Start(ctx, "PerforatorServer.renderProfile")
	defer span.End()
	defer func() {
		if err != nil {
			span.SetStatus(otelcodes.Error, err.Error())
			span.RecordError(err)
		}
	}()

	profile, err = s.symbolizeProfile(ctx, profile, format.GetSymbolize())
	if err != nil {
		return nil, err
	}

	{
		postprocessResults := interpreterstack.Postprocess(profile, interpreterstack.OptionsFromProto(format.Postprocessing)...)
		if len(postprocessResults.Python.Errors) > 0 {
			s.l.Warn(ctx, "Found errors on joining python and native stacks", log.Error(errors.Join(postprocessResults.Python.Errors...)))
		}

		s.metrics.mergedPythonStacks.Add(int64(postprocessResults.Python.MergedStacksCount))
		s.metrics.unmergedPythonStacks.Add(int64(postprocessResults.Python.UnmergedStacksCount))
		if postprocessResults.Python.MergedStacksCount+postprocessResults.Python.UnmergedStacksCount > 0 {
			s.metrics.mergedPythonStacksRatios.RecordValue(
				float64(postprocessResults.Python.MergedStacksCount) / float64(postprocessResults.Python.MergedStacksCount+postprocessResults.Python.UnmergedStacksCount),
			)
		}
	}

	start := time.Now()
	defer func() {
		if err == nil {
			s.metrics.flamegraphBuildTimer.RecordDuration(time.Since(start))
		}
	}()

	return RenderProfile(ctx, profile, format)
}

func (s *PerforatorServer) maybeUploadProfileWithID(ctx context.Context, profile []byte, id string) (url string, err error) {
	ctx, span := otel.Tracer("APIProxy").Start(ctx, "PerforatorServer.maybeUploadProfile")
	defer span.End()
	defer func() {
		if err != nil {
			span.SetStatus(otelcodes.Error, err.Error())
			span.RecordError(err)
		}
	}()

	if s.renderedProfiles == nil {
		return "", nil
	}

	w, err := s.renderedProfiles.Put(ctx, id)
	if err != nil {
		return "", fmt.Errorf("failed to start rendered profile writer: %w", err)
	}

	_, err = w.Write(profile)
	if err != nil {
		return "", fmt.Errorf("failed to upload rendered profile: %w", err)
	}

	_, err = w.Commit()
	if err != nil {
		return "", fmt.Errorf("failed to commit rendered profile upload: %w", err)
	}

	return fmt.Sprintf("%s%s", s.c.RenderedProfiles.URLPrefix, id), nil
}

func (s *PerforatorServer) maybeUploadProfile(ctx context.Context, profile []byte, format *perforator.RenderFormat) (url string, err error) {
	id, err := s.makeProfileKey(format)
	if err != nil {
		return "", err
	}

	return s.maybeUploadProfileWithID(ctx, profile, id)
}

func (s *PerforatorServer) makeProfileKey(format *perforator.RenderFormat) (string, error) {
	uid, err := uuid.NewV7()
	if err != nil {
		return "", err
	}

	suffix := ""

	switch v := format.GetFormat().(type) {
	case *perforator.RenderFormat_RawProfile:
		suffix = ".pb.gz"
	case *perforator.RenderFormat_JSONFlamegraph:
		suffix = ".json"
	case *perforator.RenderFormat_Flamegraph, *perforator.RenderFormat_HTMLVisualisation:
		suffix = ".html"
	case *perforator.RenderFormat_TextProfile:
		suffix = ".txt"
	default:
		return "", fmt.Errorf("unsupported render format: %T", v)
	}

	return uid.String() + suffix, nil
}

func (s *PerforatorServer) makePGOProfileKey(format *perforator.PGOProfileFormat) (string, error) {
	uid, err := uuid.NewV7()
	if err != nil {
		return "", err
	}

	suffix := ""
	switch v := format.GetFormat().(type) {
	case *perforator.PGOProfileFormat_AutoFDO:
		suffix = ".pgo.extbinary"
	case *perforator.PGOProfileFormat_Bolt:
		suffix = ".bolt.yaml"
	default:
		return "", fmt.Errorf("unsupported PGO render format: %T", v)
	}

	return uid.String() + suffix, nil
}
