package server

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"runtime/debug"
	"slices"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-resty/resty/v2"
	"github.com/gofrs/uuid"
	pprof "github.com/google/pprof/profile"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	otelcodes "go.opentelemetry.io/otel/codes"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health"
	healthgrpc "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/library/go/core/metrics"
	"github.com/yandex/perforator/observability/lib/querylang"
	"github.com/yandex/perforator/observability/lib/querylang/operator"
	"github.com/yandex/perforator/perforator/internal/asynctask"
	"github.com/yandex/perforator/perforator/internal/symbolizer/auth"
	"github.com/yandex/perforator/perforator/internal/symbolizer/autofdo"
	"github.com/yandex/perforator/perforator/internal/symbolizer/binaryprovider/downloader"
	"github.com/yandex/perforator/perforator/internal/symbolizer/symbolize"
	"github.com/yandex/perforator/perforator/internal/xmetrics"
	"github.com/yandex/perforator/perforator/pkg/grpcutil/grpclog"
	"github.com/yandex/perforator/perforator/pkg/grpcutil/grpcmetrics"
	"github.com/yandex/perforator/perforator/pkg/polyheapprof"
	"github.com/yandex/perforator/perforator/pkg/profile/flamegraph/render"
	"github.com/yandex/perforator/perforator/pkg/profile/python"
	"github.com/yandex/perforator/perforator/pkg/profile/samplefilter"
	"github.com/yandex/perforator/perforator/pkg/profilequerylang"
	"github.com/yandex/perforator/perforator/pkg/sampletype"
	blob "github.com/yandex/perforator/perforator/pkg/storage/blob/models"
	"github.com/yandex/perforator/perforator/pkg/storage/blob/s3"
	"github.com/yandex/perforator/perforator/pkg/storage/bundle"
	"github.com/yandex/perforator/perforator/pkg/storage/microscope"
	profilestorage "github.com/yandex/perforator/perforator/pkg/storage/profile"
	"github.com/yandex/perforator/perforator/pkg/storage/profile/meta"
	"github.com/yandex/perforator/perforator/pkg/storage/util"
	"github.com/yandex/perforator/perforator/pkg/tracing"
	"github.com/yandex/perforator/perforator/pkg/xlog"
	"github.com/yandex/perforator/perforator/proto/perforator"
)

var (
	ErrFailedGetProfile = errors.New("failed to get profile")

	ValidTargetEventTypes = map[string]struct{}{
		sampletype.SampleTypeCPUCycles:   struct{}{},
		sampletype.SampleTypeLbrStacks:   struct{}{},
		sampletype.SampleTypeSignalCount: struct{}{},
		sampletype.SampleTypeWallSeconds: struct{}{},
	}
)

// TODO(itrofimow): make this 8 configurable?
const PGODegreeOfParallelism uint64 = 8

type requestsMetrics struct {
	successes metrics.Counter
	fails     metrics.Counter
}

type perforatorServerMetrics struct {
	listServicesRequest    requestsMetrics
	listProfilesRequests   requestsMetrics
	getProfileRequests     requestsMetrics
	mergeProfilesRequests  requestsMetrics
	diffProfilesRequests   requestsMetrics
	uploadProfilesRequests requestsMetrics

	unmergedPythonStacks     metrics.Counter
	mergedPythonStacks       metrics.Counter
	mergedPythonStacksRatios metrics.Histogram

	flamegraphBuildTimer metrics.Timer
	mergeProfilesTimer   metrics.Timer

	tasksRunningCount  metrics.IntGaugeVec
	tasksStartedCount  metrics.CounterVec
	tasksFinishedCount metrics.CounterVec
	tasksFailedCount   metrics.CounterVec
}

type Tags = map[string]string

type PerforatorServer struct {
	l    xlog.Logger
	c    *Config
	reg  xmetrics.Registry
	auth *auth.Provider

	microscopeStorage microscope.Storage
	profileStorage    profilestorage.Storage
	renderedProfiles  blob.Storage
	bannedUsers       *BannedUsersRegistry
	tasks             asynctask.TaskService
	tasksemaphore     *semaphore.Weighted

	downloader *downloader.Downloader
	httpclient *resty.Client

	mutex      sync.Mutex
	symbolizer *symbolize.Symbolizer

	llvmTools LLVMTools

	grpcServer   *grpc.Server
	healthServer *health.Server
	otelShutdown func(context.Context) error
	httpRouter   chi.Router

	metrics *perforatorServerMetrics
}

func getSymbolizationMode(conf *Config) symbolize.SymbolizationMode {
	if conf.SymbolizationConfig.UseGSYM {
		return symbolize.SymbolizationModeGSYMPreferred
	} else {
		return symbolize.SymbolizationModeDWARF
	}
}

func NewPerforatorServer(
	conf *Config,
	l xlog.Logger,
	reg xmetrics.Registry,
) (server *PerforatorServer, err error) {
	conf.FillDefault()

	ctx := context.Background()

	// Setup OpenTelemetry tracing.
	exporter, err := tracing.NewExporter(ctx, conf.Tracing)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize tracing span exporter: %w", err)
	}

	shutdown, _, err := tracing.Initialize(ctx, l.WithName("tracing").Logger(), exporter, "perforator", "proxy")
	if err != nil {
		return nil, fmt.Errorf("failed to initialize tracing: %w", err)
	}
	defer func() {
		if err != nil && shutdown != nil {
			_ = shutdown(ctx)
		}
	}()
	l.Info(ctx, "Successfully initialized tracing")

	initCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	storageBundle, err := bundle.NewStorageBundle(initCtx, l, reg, &conf.StorageConfig)
	if err != nil {
		return nil, err
	}
	l.Info(ctx, "Initialized storage bundle")

	var renderedProfiles blob.Storage
	if conf.RenderedProfiles != nil {
		if storageBundle.DBs.S3Client == nil {
			return nil, errors.New("s3 is not specified")
		}

		renderedProfiles, err = s3.NewS3Storage(l, reg.WithPrefix("rendered_profiles_storage"), storageBundle.DBs.S3Client, conf.RenderedProfiles.S3Bucket)
		if err != nil {
			return nil, fmt.Errorf("failed to create rendered profiles storage: %w", err)
		}
	}

	downloaderInstance, binaryDownloader, gsymDownloader, err := downloader.CreateDownloaders(
		conf.BinaryProvider.FileCache,
		conf.BinaryProvider.MaxSimultaneousDownloads,
		l, reg,
		storageBundle.BinaryStorage.Binary(), storageBundle.BinaryStorage.GSYM(),
	)

	llvmTools := NewLLVMTools(
		l.WithName("llvmTools"),
		conf.PGOConfig,
		binaryDownloader,
	)

	symbolizer, err := symbolize.NewSymbolizer(
		l.WithName("symbolizer"),
		reg,
		binaryDownloader,
		gsymDownloader,
		getSymbolizationMode(conf),
	)
	if err != nil {
		return nil, err
	}

	authp, err := newAuthProvider(l, conf.Server.Insecure)
	if err != nil {
		return nil, err
	}
	oauthInterceptor := authp.GRPC([]string{healthgrpc.Health_Watch_FullMethodName, healthgrpc.Health_Check_FullMethodName})

	logInterceptor := grpclog.
		NewLogInterceptor(l.WithName("grpc")).
		SkipMethods(healthgrpc.Health_Watch_FullMethodName).
		SkipMethods(healthgrpc.Health_Check_FullMethodName)

	metricsInterceptor := grpcmetrics.NewMetricsInterceptor(reg)

	grpcServer := grpc.NewServer(
		grpc.MaxSendMsgSize(1024*1024*1024 /*1G*/),
		grpc.MaxRecvMsgSize(1024*1024*1024 /*1G*/),
		grpc.KeepaliveEnforcementPolicy(
			keepalive.EnforcementPolicy{
				MinTime: 20 * time.Second,
			},
		),
		grpc.ChainUnaryInterceptor(
			otelgrpc.UnaryServerInterceptor(),
			metricsInterceptor.UnaryServer(),
			logInterceptor.UnaryServer(),
			oauthInterceptor.UnaryServer(),
		),
		grpc.ChainStreamInterceptor(
			otelgrpc.StreamServerInterceptor(),
			metricsInterceptor.StreamServer(),
			logInterceptor.StreamServer(),
			oauthInterceptor.StreamServer(),
		),
	)

	httpr := chi.NewRouter()
	httpr.Use(middleware.Recoverer)
	httpr.Use(otelhttp.NewMiddleware("http.server"))
	httpr.Use(authp.HTTP())

	healthServer := health.NewServer()
	healthgrpc.RegisterHealthServer(grpcServer, healthServer)

	server = &PerforatorServer{
		l:                 l,
		c:                 conf,
		reg:               reg,
		microscopeStorage: storageBundle.MicroscopeStorage,
		profileStorage:    storageBundle.ProfileStorage,
		renderedProfiles:  renderedProfiles,
		bannedUsers:       NewBannedUsersRegistry(ctx, l, reg, storageBundle.DBs),
		tasks:             storageBundle.TaskStorage,
		tasksemaphore:     semaphore.NewWeighted(conf.Tasks.ConcurrencyLimit),
		httpclient:        resty.New().SetTimeout(time.Hour).SetRetryCount(3),
		downloader:        downloaderInstance,
		llvmTools:         llvmTools,
		symbolizer:        symbolizer,
		grpcServer:        grpcServer,
		httpRouter:        httpr,
		healthServer:      healthServer,
		otelShutdown:      shutdown,
	}

	mux := runtime.NewServeMux()
	err = errors.Join(
		perforator.RegisterPerforatorHandlerServer(ctx, mux, server),
		perforator.RegisterTaskServiceHandlerServer(ctx, mux, server),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to register grpc-gateway server: %w", err)
	}
	server.httpRouter.Mount("/", mux)

	perforator.RegisterPerforatorServer(server.grpcServer, server)
	perforator.RegisterTaskServiceServer(server.grpcServer, server)
	perforator.RegisterMicroscopeServiceServer(server.grpcServer, server)
	reflection.Register(server.grpcServer)

	server.registerMetrics()

	return server, nil
}

func (s *PerforatorServer) registerMetrics() {
	s.metrics = &perforatorServerMetrics{
		listProfilesRequests: requestsMetrics{
			successes: s.reg.WithTags(Tags{"status": "success"}).Counter("requests.list_profiles"),
			fails:     s.reg.WithTags(Tags{"status": "fail"}).Counter("requests.list_profiles"),
		},
		listServicesRequest: requestsMetrics{
			successes: s.reg.WithTags(Tags{"status": "success"}).Counter("requests.list_services"),
			fails:     s.reg.WithTags(Tags{"status": "fail"}).Counter("requests.list_services"),
		},
		getProfileRequests: requestsMetrics{
			successes: s.reg.WithTags(Tags{"status": "success"}).Counter("requests.get_profile"),
			fails:     s.reg.WithTags(Tags{"status": "fail"}).Counter("requests.get_profile"),
		},
		mergeProfilesRequests: requestsMetrics{
			successes: s.reg.WithTags(Tags{"status": "success"}).Counter("requests.merge_profiles"),
			fails:     s.reg.WithTags(Tags{"status": "fail"}).Counter("requests.merge_profiles"),
		},
		diffProfilesRequests: requestsMetrics{
			successes: s.reg.WithTags(Tags{"status": "success"}).Counter("requests.diff_profiles"),
			fails:     s.reg.WithTags(Tags{"status": "fail"}).Counter("requests.diff_profiles"),
		},
		uploadProfilesRequests: requestsMetrics{
			successes: s.reg.WithTags(Tags{"status": "success"}).Counter("requests.upload_profile"),
			fails:     s.reg.WithTags(Tags{"status": "fail"}).Counter("requests.upload_profile"),
		},
		flamegraphBuildTimer: s.reg.Timer("flamegraph.build"),
		mergeProfilesTimer:   s.reg.Timer("profile.merge"),

		unmergedPythonStacks: s.reg.WithTags(Tags{"result": "ok"}).Counter("python.merge_stacks.count"),
		mergedPythonStacks:   s.reg.WithTags(Tags{"result": "fail"}).Counter("python.merge_stacks.count"),
		mergedPythonStacksRatios: s.reg.Histogram(
			"python.merge_stacks.ratio.hist",
			metrics.MakeLinearBuckets(0, float64(0.02), 50),
		),

		tasksRunningCount:  s.reg.IntGaugeVec("tasks.running.count", []string{"kind"}),
		tasksStartedCount:  s.reg.CounterVec("tasks.started.count", []string{"kind"}),
		tasksFinishedCount: s.reg.CounterVec("tasks.finished.count", []string{"kind"}),
		tasksFailedCount:   s.reg.CounterVec("tasks.failed.count", []string{"kind"}),
	}
}

////////////////////////////////////////////////////////////////////////////////

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
		err = errors.New("prefix is not supported yet")
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
		return nil, err
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
		return nil, err
	}

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
		return nil, err
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
	}
}

func extractProtoMetasFromRawProfiles(profiles []*profilestorage.Profile) []*perforator.ProfileMeta {
	protometas := make([]*perforator.ProfileMeta, len(profiles))
	for i, profile := range profiles {
		protometas[i] = storageMetaToProtoMeta(profile.Meta)
	}
	return protometas
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

	selector.Matchers = append(
		selector.Matchers,
		s.buildExcludeProfilerVersionMatcher(),
	)

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
		return nil, fmt.Errorf("failed to parse profile query: %w", err)
	}

	query.Pagination.Limit = uint64(req.GetPaginated().GetLimit())
	query.Pagination.Offset = uint64(req.GetPaginated().GetOffset())
	query.SortOrder = util.SortOrderFromProto(req.GetOrderBy())

	var profiles []*profilestorage.Profile
	profiles, err = s.profileStorage.SelectProfiles(ctx, query, true)
	if err != nil {
		return nil, err
	}

	metas := make([]*perforator.ProfileMeta, 0, len(profiles))
	for _, profile := range profiles {
		metas = append(
			metas,
			storageMetaToProtoMeta(profile.Meta),
		)
	}

	return &perforator.ListProfilesResponse{
		Profiles: metas,
	}, nil
}

func (s *PerforatorServer) fetchProfile(ctx context.Context, id meta.ProfileID) (profile *pprof.Profile, meta *meta.ProfileMetadata, err error) {
	ctx, span := otel.Tracer("APIProxy").Start(ctx, "PerforatorServer.fetchProfile")
	defer span.End()
	defer func() {
		if err != nil {
			span.SetStatus(otelcodes.Error, err.Error())
			span.RecordError(err)
		}
	}()

	var rawProfiles []*profilestorage.Profile
	rawProfiles, err = s.profileStorage.GetProfiles(ctx, []string{id} /*onlyMetadata=*/, false)
	if err != nil {
		return
	}

	if len(rawProfiles) != 1 {
		err = ErrFailedGetProfile
		return
	}

	profile, err = pprof.ParseData(rawProfiles[0].Body)
	if err != nil {
		return
	}

	meta = rawProfiles[0].Meta
	return
}

func (s *PerforatorServer) guessBuildIDForPGO(ctx context.Context, rawProfiles []*profilestorage.Profile) (string, error) {
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
			for j := guesserIndex; j < uint64(len(rawProfiles)); j += PGODegreeOfParallelism {
				err := guesser.FeedProfile(guesserIndex, rawProfiles[j].Body)
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
	rawProfiles []*profilestorage.Profile,
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
			for j := builderIndex; j < uint64(len(rawProfiles)); j += PGODegreeOfParallelism {
				err := builder.AddProfile(builderIndex, rawProfiles[j].Body)
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
	service string,
	maxSamples uint32,
	profilesToProcessTotalSizeLimit uint64,
) (autofdoInput, error) {
	selector := fmt.Sprintf("{%s=\"%s\", %s=\"%s\"}",
		profilequerylang.EventTypeLabel, sampletype.SampleTypeLbrStacks,
		profilequerylang.ServiceLabel, service)

	query, err := s.parseProfileQuery(&perforator.ProfileQuery{
		Selector: selector,
		TimeInterval: &perforator.TimeInterval{
			// Given that we _guess_ the target buildID, it makes sense to look for
			// somewhat recent profiles only.
			From: timestamppb.New(time.Now().Add(-time.Hour * 24)),
		},
	})
	if err != nil {
		return autofdoInput{}, err
	}
	// Again, this increases the chance of guessing a buildID from the most recent release,
	// instead of the previous one(s).
	query.SortOrder = util.SortOrder{
		Columns:    []string{profilequerylang.TimestampLabel},
		Descending: true,
	}
	query.MaxSamples = 0
	query.Pagination = util.Pagination{
		Offset: 0,
		Limit:  uint64(maxSamples),
	}
	rawProfiles, err := s.selectProfilesLimited(ctx, query, profilesToProcessTotalSizeLimit)
	if err != nil {
		return autofdoInput{}, err
	}
	rawProfiles = filterNoBlobProfiles(rawProfiles)

	buildID, err := s.guessBuildIDForPGO(ctx, rawProfiles)
	if err != nil {
		return autofdoInput{}, err
	}

	processesLBR, err := s.processLBRProfiles(ctx, rawProfiles, buildID)
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
	service string,
	format *perforator.PGOProfileFormat,
	maxSamples uint32,
	profilesToProcessTotalSizeLimit uint64,
) ([]byte, *perforator.PGOMeta, error) {
	autofdoInput, err := s.generateAutofdoInput(ctx, service, maxSamples, profilesToProcessTotalSizeLimit)
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
	}, nil
}

// A subroutine of MergeProfiles
func (s *PerforatorServer) fetchProfiles(
	ctx context.Context,
	query *meta.ProfileQuery,
	targetEventType string,
) (*pprof.Profile, []*profilestorage.Profile, error) {
	var rawProfiles []*profilestorage.Profile
	rawProfiles, err := s.selectProfiles(ctx, query)
	if err != nil {
		return nil, nil, err
	}

	rawProfiles = filterNoBlobProfiles(rawProfiles)

	profiles, err := s.parseProfiles(ctx, rawProfiles)
	if err != nil {
		return nil, nil, err
	}

	tlsFilter, err := samplefilter.BuildTLSFilter(query.Selector)
	if err != nil {
		return nil, nil, err
	}

	envFilter, envErr := samplefilter.BuildEnvFilter(query.Selector)
	if envErr != nil {
		return nil, nil, envErr
	}

	buildIDFilter, err := samplefilter.BuildBuildIDFilter(query.Selector)
	if err != nil {
		return nil, nil, err
	}

	postprocessedProfiles := samplefilter.FilterProfilesBySampleFilters(
		profiles,
		tlsFilter,
		envFilter,
		buildIDFilter,
	)

	for _, profile := range postprocessedProfiles {
		fixupMultiSampleTypeProfile(profile, targetEventType)
	}

	mergedProfile, err := s.mergeProfiles(ctx, postprocessedProfiles)
	if err != nil {
		return nil, nil, err
	}

	return mergedProfile, rawProfiles, nil
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
		return nil, fmt.Errorf("failed to fetch profiles: %w", err)
	}

	// TODO: specify target event type in GetProfileRequest
	targetEventType := meta.MainEventType
	if targetEventType != sampletype.SampleTypeLbrStacks {
		targetEventType = sampletype.SampleTypeCPUCycles
	}
	fixupMultiSampleTypeProfile(profile, targetEventType)

	if meta.MainEventType == sampletype.SampleTypeLbrStacks && !isRawRenderFormat(req.GetFormat()) {
		return nil, fmt.Errorf("only RawProfile format is supported for this profile")
	}

	l.Info(ctx, "Rendering profile")
	var buf []byte
	buf, err = s.renderProfile(ctx, profile, req.GetFormat())
	if err != nil {
		return nil, fmt.Errorf("failed to render profile: %w", err)
	}

	return &perforator.GetProfileResponse{
		Profile:     buf,
		ProfileMeta: storageMetaToProtoMeta(meta),
	}, nil
}

func filterNoBlobProfiles(profiles []*profilestorage.Profile) (res []*profilestorage.Profile) {
	res = make([]*profilestorage.Profile, 0, len(profiles))
	for _, profile := range profiles {
		if len(profile.Body) > 0 {
			res = append(res, profile)
		}
	}

	return
}

func (s *PerforatorServer) GeneratePGOProfile(
	ctx context.Context,
	req *perforator.GeneratePGOProfileRequest,
) (*perforator.GeneratePGOProfileResponse, error) {
	if len(req.GetService()) == 0 {
		return nil, fmt.Errorf("empty \"service\"")
	}

	// We deal with large binaries and we want A LOT of lbr-profiles
	// for better resulting sPGO-profile quality. This is basically a
	// "number big enough".
	const maxSamples = uint32(5000)
	// Some services are loaded heavier than others, and thus generate larger profiles.
	// We don't want to run out of memory when downloading 5k of huge profiles
	// (10Mb per-profile is not uncommon, which would amount to whopping 50GB of data).
	// Presumably, 8Gb of data should be more than enough, and we would either hit
	// "maxSamples" limit for not-that-heavy-loaded services, or this one.
	const profilesToProcessTotalSizeLimit = uint64(8 * 1024 * 1024 * 1024)
	buf, PGOmeta, err := s.doGeneratePGOProfile(ctx, req.GetService(), req.GetFormat(), maxSamples, profilesToProcessTotalSizeLimit)
	if err != nil {
		return nil, err
	}

	id, err := s.makePGOProfileKey(req.GetFormat())
	if err != nil {
		return nil, err
	}

	url, err := s.maybeUploadProfileWithID(ctx, buf, id)
	if err != nil {
		return nil, err
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

// MergeProfiles implements perforator.PerforatorServer
func (s *PerforatorServer) MergeProfiles(
	ctx context.Context,
	req *perforator.MergeProfilesRequest,
) (*perforator.MergeProfilesResponse, error) {
	var err error
	defer func() {
		if err != nil {
			s.metrics.mergeProfilesRequests.fails.Inc()
		} else {
			s.metrics.mergeProfilesRequests.successes.Inc()
		}
	}()

	query, err := s.parseProfileQuery(req.GetQuery())
	if err != nil {
		return nil, fmt.Errorf("failed to parse profile query: %w", err)
	}

	targetEventType, err := deriveEventTypeFromSelector(query.Selector)
	if err != nil {
		return nil, fmt.Errorf("failed to derive target event type from selector: %w", err)
	}

	// some validation here
	if _, ok := ValidTargetEventTypes[targetEventType]; !ok {
		return nil, fmt.Errorf("event type %s is not valid", targetEventType)
	}

	if query.MaxSamples == 0 {
		query.MaxSamples = uint64(req.MaxSamples)
	}
	mergedProfile, rawProfiles, err := s.fetchProfiles(ctx, query, targetEventType)
	if err != nil {
		return nil, err
	}

	buf, err := s.renderProfile(ctx, mergedProfile, req.GetFormat())
	if err != nil {
		return nil, err
	}

	url, err := s.maybeUploadProfile(ctx, buf, req.GetFormat())
	if err != nil {
		return nil, err
	}

	if url != "" {
		return &perforator.MergeProfilesResponse{
			Result:      &perforator.MergeProfilesResponse_ProfileURL{ProfileURL: url},
			ProfileMeta: extractProtoMetasFromRawProfiles(rawProfiles),
		}, nil
	} else {
		return &perforator.MergeProfilesResponse{
			Result:      &perforator.MergeProfilesResponse_Profile{Profile: buf},
			ProfileMeta: extractProtoMetasFromRawProfiles(rawProfiles),
		}, nil
	}
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
			fgOptions = req.RenderFormat.GetFlamegraph()
			fgFormat = render.HTMLFormat
		case *perforator.RenderFormat_JSONFlamegraph:
			fgOptions = req.RenderFormat.GetJSONFlamegraph()
			fgFormat = render.JSONFormat
		default:
			return nil, fmt.Errorf("unsupported diff render format %T", v)
		}
	}

	err = fillFlamegraphOptions(fg, fgOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to fill flamegraph options: %w", err)
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
		return nil, err
	}

	var buf []byte
	buf, err = fg.RenderBytes()
	if err != nil {
		return nil, err
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
		return nil, err
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
					Query:      query,
					MaxSamples: query.GetMaxSamples(),
				},
			},
		},
	})

	if err != nil {
		return "", err
	}

	return task.TaskID, nil
}

const defaultMaxSamples = 10

func (s *PerforatorServer) selectProfilesLimited(
	ctx context.Context,
	filters *meta.ProfileQuery,
	batchDownloadTotalSizeSoftLimit uint64,
) (profiles []*profilestorage.Profile, err error) {
	ctx, span := otel.Tracer("APIProxy").Start(ctx, "PerforatorServer.selectProfiles")
	defer span.End()
	defer func() {
		if err != nil {
			span.SetStatus(otelcodes.Error, err.Error())
			span.RecordError(err)
		}
	}()

	if filters.MaxSamples == 0 && filters.Limit == 0 {
		filters.MaxSamples = defaultMaxSamples
	}

	return s.profileStorage.SelectProfilesLimited(
		ctx,
		filters,
		batchDownloadTotalSizeSoftLimit,
	)
}

func (s *PerforatorServer) selectProfiles(
	ctx context.Context,
	filters *meta.ProfileQuery,
) (profiles []*profilestorage.Profile, err error) {
	return s.selectProfilesLimited(ctx, filters, profilestorage.DefaultBatchDownloadTotalSizeSoftLimit)
}

func (s *PerforatorServer) parseProfile(ctx context.Context, rawProfile *profilestorage.Profile) (profile *pprof.Profile, err error) {
	_, span := otel.Tracer("APIProxy").Start(ctx, "PerforatorServer.parseProfile")
	defer span.End()
	defer func() {
		if err != nil {
			span.SetStatus(otelcodes.Error, err.Error())
			span.RecordError(err)
		}
	}()

	profile, err = pprof.ParseData(rawProfile.Body)
	return
}

func (s *PerforatorServer) parseProfiles(
	ctx context.Context,
	rawProfiles []*profilestorage.Profile,
) (profiles []*pprof.Profile, err error) {
	ctx, span := otel.Tracer("APIProxy").Start(ctx, "PerforatorServer.parseProfiles")
	defer span.End()
	defer func() {
		if err != nil {
			span.SetStatus(otelcodes.Error, err.Error())
			span.RecordError(err)
		}
	}()

	profiles = make([]*pprof.Profile, len(rawProfiles))

	g, ctx := errgroup.WithContext(ctx)

	for i, rawProfile := range rawProfiles {
		i := i
		rawProfile := rawProfile

		g.Go(func() error {
			var errParse error
			profiles[i], errParse = s.parseProfile(ctx, rawProfile)
			return errParse
		})
	}

	err = g.Wait()
	if err != nil {
		return
	}

	return
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

	if format.Postprocessing == nil || format.Postprocessing.MergePythonAndNativeStacks == nil || *format.Postprocessing.MergePythonAndNativeStacks {
		postprocessResults := python.PostprocessSymbolizedProfileWithPython(profile)
		if len(postprocessResults.Errors) > 0 {
			s.l.Warn(ctx, "Found errors on joining python and native stacks", log.Error(errors.Join(postprocessResults.Errors...)))
		}

		s.metrics.mergedPythonStacks.Add(int64(postprocessResults.MergedStacksCount))
		s.metrics.unmergedPythonStacks.Add(int64(postprocessResults.UnmergedStacksCount))
		if postprocessResults.MergedStacksCount+postprocessResults.UnmergedStacksCount > 0 {
			s.metrics.mergedPythonStacksRatios.RecordValue(
				float64(postprocessResults.MergedStacksCount) / float64(postprocessResults.MergedStacksCount+postprocessResults.UnmergedStacksCount),
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
	case *perforator.RenderFormat_Flamegraph:
		suffix = ".html"
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

////////////////////////////////////////////////////////////////////////////////

type RunConfig struct {
	MetricsPort uint32
	HTTPPort    uint32
	GRPCPort    uint32
}

func (s *PerforatorServer) runMetricsServer(ctx context.Context, port uint32) error {
	s.l.Info(ctx, "Starting metrics server", log.UInt32("port", port))
	http.Handle("/metrics", s.reg.HTTPHandler(ctx, s.l))
	http.HandleFunc("/debug/pprof/polyheap", func(w http.ResponseWriter, r *http.Request) {
		p, err := polyheapprof.ReadCurrentHeapProfile()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/octet-stream")
		_ = p.Write(w)
	})
	return http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
}

func (s *PerforatorServer) runGRPCServer(ctx context.Context, port uint32) error {
	s.l.Info(ctx, "Starting profile storage server", log.UInt32("port", port))

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return err
	}

	// set healthy status for whole system and for Perforator service
	s.healthServer.SetServingStatus("", healthgrpc.HealthCheckResponse_SERVING)
	s.healthServer.SetServingStatus("NPerforator.NProto.Perforator", healthgrpc.HealthCheckResponse_SERVING)

	return s.grpcServer.Serve(lis)
}

func (s *PerforatorServer) runHTTPServer(ctx context.Context, port uint32) error {
	s.l.Info(ctx, "Starting HTTP REST server on port", log.UInt32("port", port))
	srv := &http.Server{Addr: fmt.Sprintf(":%d", port), Handler: s.httpRouter}
	return srv.ListenAndServe()
}

func (s *PerforatorServer) Run(ctx context.Context, conf *RunConfig) error {
	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		err := s.downloader.RunBackgroundDownloader(context.Background())
		if err != nil {
			s.l.Error(ctx, "Failed background downloader", log.Error(err))
		}
		return err
	})

	g.Go(func() error {
		err := s.runMetricsServer(ctx, conf.MetricsPort)
		if err != nil {
			s.l.Error(ctx, "Failed metrics server", log.Error(err))
		}
		return err
	})

	g.Go(func() error {
		err := s.runGRPCServer(ctx, conf.GRPCPort)
		if err != nil {
			s.l.Error(ctx, "GRPC server failed", log.Error(err))
		}
		return err
	})

	g.Go(func() error {
		err := s.runHTTPServer(ctx, conf.HTTPPort)
		if err != nil {
			s.l.Error(ctx, "HTTP server failed", log.Error(err))
		}
		return err
	})

	g.Go(func() error {
		err := s.runAsyncTasks(ctx)
		if err != nil {
			s.l.Error(ctx, "Async tasks runner failed", log.Error(err))
		}
		return err
	})

	return g.Wait()
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

	return pprof.Merge(profiles)
}

func cleanupTransientLabels(profile *pprof.Profile) error {
	for _, sample := range profile.Sample {
		delete(sample.Label, "cgroup")
		delete(sample.NumLabel, "pid")
		delete(sample.NumLabel, "tid")
	}
	return nil
}

////////////////////////////////////////////////////////////////////////////////

const (
	uploadProfileSystemName = "uploads"
)

// UploadProfile implements perforator.PerforatorServer.
func (s *PerforatorServer) UploadProfile(ctx context.Context, req *perforator.UploadProfileRequest) (*perforator.UploadProfileResponse, error) {
	var err error
	defer func() {
		if err != nil {
			s.metrics.uploadProfilesRequests.fails.Inc()
		} else {
			s.metrics.uploadProfilesRequests.successes.Inc()
		}
	}()

	// Try to parse profile slowly in order to check it validity.
	var profile *pprof.Profile
	profile, err = pprof.ParseData(req.GetProfile())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to parse profile: %v", err)
	}

	var metadata *meta.ProfileMetadata
	metadata, err = makeUploadProfileMeta(ctx, req.GetProfileMeta(), profile)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to parse metadata: %v", err)
	}
	s.l.Info(ctx, "Uploading profile",
		log.Any("meta", metadata),
		log.Int("bytesize", len(req.GetProfile())),
	)

	metas := denormalizeProfileMeta(metadata)
	var id string
	id, err = s.profileStorage.StoreProfile(ctx, metas, req.GetProfile())
	if err != nil {
		s.l.Error(ctx, "Failed to upload profile", log.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to store profile: %v", err)
	}
	s.l.Info(ctx, "Successfully uploaded profile", log.String("id", id))

	return &perforator.UploadProfileResponse{ProfileID: id}, nil
}

func makeUploadProfileMeta(ctx context.Context, protometa *perforator.ProfileMeta, profile *pprof.Profile) (*meta.ProfileMetadata, error) {
	res := &meta.ProfileMetadata{
		System:        uploadProfileSystemName,
		AllEventTypes: protometa.GetEventTypes(),
		Cluster:       protometa.GetCluster(),
		Service:       protometa.GetService(),
		PodID:         protometa.GetPodID(),
		NodeID:        protometa.GetNodeID(),
		Timestamp:     protometa.GetTimestamp().AsTime(),
		BuildIDs:      protometa.GetBuildIDs(),
		Attributes:    protometa.GetAttributes(),
	}

	if len(res.AllEventTypes) == 0 {
		if len(profile.SampleType) > 0 {
			for _, eventType := range profile.SampleType {
				res.AllEventTypes = append(res.AllEventTypes, sampletype.SampleTypeToString(eventType))
			}
		} else if eventType := protometa.GetEventType(); eventType != "" {
			res.AllEventTypes = append(res.AllEventTypes, eventType)
		} else {
			return nil, status.Errorf(codes.InvalidArgument, "malformed profile metadata: no event type found")
		}
	}

	if res.Timestamp.IsZero() {
		if profile.TimeNanos != 0 {
			res.Timestamp = time.UnixMicro(profile.TimeNanos / 1000)
		} else {
			res.Timestamp = time.Now()
		}
	}

	if len(res.BuildIDs) == 0 {
		for _, mapping := range profile.Mapping {
			if mapping == nil {
				continue
			}

			if mapping.BuildID != "" {
				res.BuildIDs = append(res.BuildIDs, mapping.BuildID)
			}
		}
	}

	if res.Attributes == nil {
		res.Attributes = make(map[string]string)
	}
	if user := auth.UserFromContext(ctx); user != nil {
		res.Attributes["author"] = user.Login
	}

	res.Attributes["format"] = "pprof"
	res.Attributes["origin"], _ = os.Hostname()

	return res, nil
}

func denormalizeProfileMeta(commonMeta *meta.ProfileMetadata) []*meta.ProfileMetadata {
	metas := make([]*meta.ProfileMetadata, 0, len(commonMeta.AllEventTypes))
	for _, eventType := range commonMeta.AllEventTypes {
		meta := *commonMeta
		meta.MainEventType = eventType
		metas = append(metas, &meta)
	}
	return metas
}

////////////////////////////////////////////////////////////////////////////////
