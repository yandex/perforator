package server

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-resty/resty/v2"
	pprof "github.com/google/pprof/profile"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health"
	healthgrpc "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/library/go/core/metrics"
	"github.com/yandex/perforator/perforator/internal/asynctask"
	bpclient "github.com/yandex/perforator/perforator/internal/binaryprocessor/client"
	"github.com/yandex/perforator/perforator/internal/symbolizer/auth"
	"github.com/yandex/perforator/perforator/internal/symbolizer/binaryprovider/downloader"
	"github.com/yandex/perforator/perforator/internal/symbolizer/proxy/services"
	"github.com/yandex/perforator/perforator/internal/symbolizer/symbolize"
	"github.com/yandex/perforator/perforator/internal/xmetrics"
	"github.com/yandex/perforator/perforator/pkg/cprofile"
	"github.com/yandex/perforator/perforator/pkg/grpcutil/grpclog"
	"github.com/yandex/perforator/perforator/pkg/grpcutil/grpcmetrics"
	"github.com/yandex/perforator/perforator/pkg/polyheapprof"
	"github.com/yandex/perforator/perforator/pkg/sampletype"
	blob "github.com/yandex/perforator/perforator/pkg/storage/blob/models"
	"github.com/yandex/perforator/perforator/pkg/storage/blob/s3"
	"github.com/yandex/perforator/perforator/pkg/storage/bundle"
	clustertop "github.com/yandex/perforator/perforator/pkg/storage/cluster_top"
	"github.com/yandex/perforator/perforator/pkg/storage/microscope"
	profilestorage "github.com/yandex/perforator/perforator/pkg/storage/profile"
	"github.com/yandex/perforator/perforator/pkg/storage/profile/meta"
	"github.com/yandex/perforator/perforator/pkg/tracing"
	"github.com/yandex/perforator/perforator/pkg/xlog"
	"github.com/yandex/perforator/perforator/proto/perforator"
)

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
	// From enqueue to finish
	tasksProcessingSucceededDuration metrics.TimerVec
	// Same, but for failed tasks
	tasksProcessingFailedDuration metrics.TimerVec
	// From dequeue to finish
	tasksExecutionSucceededDuration metrics.TimerVec
	// Same, but for failed tasks
	tasksExecutionFailedDuration metrics.TimerVec
	// From enqueue to dequeue
	tasksWaitDuration metrics.TimerVec

	remoteSymbolizationCount             requestsMetrics
	remoteSymbolizationCompletenessCount requestsMetrics

	symbolizationProxyFallbackCount metrics.Counter
}

type Tags = map[string]string

type PerforatorServer struct {
	l    xlog.Logger
	c    *Config
	reg  xmetrics.Registry
	auth *auth.Provider

	microscopeStorage           microscope.Storage
	profileStorage              profilestorage.Storage
	clusterTopGenerationStorage clustertop.Storage
	renderedProfiles            blob.Storage
	bannedUsers                 *BannedUsersRegistry
	tasks                       asynctask.TaskService
	tasksemaphore               *semaphore.Weighted

	downloader *downloader.Downloader
	httpclient *resty.Client

	symbolizer   *symbolize.Symbolizer
	mergemanager *cprofile.MergeManager

	bpClient *bpclient.DistributingClient

	llvmTools LLVMTools

	grpcServer   *grpc.Server
	healthServer *health.Server
	otelShutdown func(context.Context) error
	httpRouter   chi.Router

	metrics *perforatorServerMetrics

	// TODO: Later all grpc services can be aggregated here
	// e.g: PerforatorService, TaskService, MicroscopeService, ReflectionService, ...
	additionalGrpcServices []services.GRPCService
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

	// TODO: this context should be tied to e.g. Run() duration.
	bgCtx := context.TODO()

	storageBundle, err := bundle.NewStorageBundle(initCtx, bgCtx, l, "proxy", reg, &conf.StorageConfig)
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

	bannedUsers, err := NewBannedUsersRegistry(ctx, l, reg, storageBundle.DBs.PostgresCluster)
	if err != nil {
		return nil, fmt.Errorf("failed to create banned users poller: %w", err)
	}

	downloaderInstance, binaryDownloader, gsymDownloader, err := downloader.CreateDownloaders(
		conf.BinaryProvider.FileCache,
		conf.BinaryProvider.MaxSimultaneousDownloads,
		l, reg,
		storageBundle.BinaryStorage.Binary(), storageBundle.BinaryStorage.GSYM(),
	)

	if err != nil {
		return nil, fmt.Errorf("failed to initialize binary downloaders: %w", err)
	}

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

	mergemanager, err := cprofile.NewMergeManager(int(conf.ProfileMerger.ThreadCount))
	if err != nil {
		return nil, err
	}

	var bpClient *bpclient.DistributingClient
	if *conf.FeaturesConfig.EnableRemoteSymbolization {
		bpClient, err = bpclient.NewDistributingClient(
			conf.BinaryProcessorClientConfig,
			l.WithName("BPClient"),
		)

		if err != nil {
			return nil, err
		}
	}

	additionalGrpcServices, err := newServices(&conf.FeaturesConfig, l, reg, storageBundle)
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
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
		grpc.ChainUnaryInterceptor(
			metricsInterceptor.UnaryServer(),
			logInterceptor.UnaryServer(),
			oauthInterceptor.UnaryServer(),
			newAccessUnaryInterceptor(conf.ACL),
		),
		grpc.ChainStreamInterceptor(
			otelgrpc.StreamServerInterceptor(),
			metricsInterceptor.StreamServer(),
			logInterceptor.StreamServer(),
			oauthInterceptor.StreamServer(),
			newAccessStreamInterceptor(),
		),
	)

	httpr := chi.NewRouter()
	httpr.Use(middleware.Recoverer)
	httpr.Use(otelhttp.NewMiddleware("http.server"))
	httpr.Use(authp.HTTP())

	healthServer := health.NewServer()
	healthgrpc.RegisterHealthServer(grpcServer, healthServer)

	server = &PerforatorServer{
		l:                           l,
		c:                           conf,
		reg:                         reg,
		microscopeStorage:           storageBundle.MicroscopeStorage,
		profileStorage:              storageBundle.ProfileStorage,
		clusterTopGenerationStorage: storageBundle.ClusterTopGenerationsStorage,
		renderedProfiles:            renderedProfiles,
		bannedUsers:                 bannedUsers,
		tasks:                       storageBundle.TaskStorage,
		tasksemaphore:               semaphore.NewWeighted(conf.Tasks.ConcurrencyLimit),
		httpclient:                  resty.New().SetTimeout(time.Hour).SetRetryCount(3),
		downloader:                  downloaderInstance,
		llvmTools:                   llvmTools,
		symbolizer:                  symbolizer,
		mergemanager:                mergemanager,
		bpClient:                    bpClient,
		grpcServer:                  grpcServer,
		httpRouter:                  httpr,
		healthServer:                healthServer,
		otelShutdown:                shutdown,
		additionalGrpcServices:      additionalGrpcServices,
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

	for _, service := range server.additionalGrpcServices {
		if err := service.Register(server.grpcServer); err != nil {
			return nil, fmt.Errorf("failed to register grpc service: %w", err)
		}
		if err := service.RegisterHandler(ctx, mux); err != nil {
			return nil, err
		}
	}

	server.registerMetrics()

	return server, nil
}

func (s *PerforatorServer) registerMetrics() {
	tasksDurationBuckets := metrics.MakeExponentialDurationBuckets(time.Millisecond*500, 1.5, 23)

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

		tasksRunningCount:                s.reg.IntGaugeVec("tasks.running.count", []string{"kind"}),
		tasksStartedCount:                s.reg.CounterVec("tasks.started.count", []string{"kind"}),
		tasksFinishedCount:               s.reg.CounterVec("tasks.finished.count", []string{"kind"}),
		tasksFailedCount:                 s.reg.CounterVec("tasks.failed.count", []string{"kind"}),
		tasksProcessingSucceededDuration: s.reg.WithTags(Tags{"status": "success"}).DurationHistogramVec("tasks.processing.duration", tasksDurationBuckets, []string{"kind"}),
		tasksProcessingFailedDuration:    s.reg.WithTags(Tags{"status": "fail"}).DurationHistogramVec("tasks.processing.duration", tasksDurationBuckets, []string{"kind"}),
		tasksExecutionSucceededDuration:  s.reg.WithTags(Tags{"status": "success"}).DurationHistogramVec("tasks.execution.duration", tasksDurationBuckets, []string{"kind"}),
		tasksExecutionFailedDuration:     s.reg.WithTags(Tags{"status": "fail"}).DurationHistogramVec("tasks.execution.duration", tasksDurationBuckets, []string{"kind"}),
		tasksWaitDuration:                s.reg.DurationHistogramVec("tasks.wait.duration", tasksDurationBuckets, []string{"kind"}),

		remoteSymbolizationCount: requestsMetrics{
			successes: s.reg.WithTags(Tags{"status": "success"}).Counter("remote_symbolization.count"),
			fails:     s.reg.WithTags(Tags{"status": "fail"}).Counter("remote_symbolization.count"),
		},
		remoteSymbolizationCompletenessCount: requestsMetrics{
			successes: s.reg.WithTags(Tags{"status": "complete"}).Counter("remote_symbolization.count"),
			fails:     s.reg.WithTags(Tags{"status": "incomplete"}).Counter("remote_symbolization.count"),
		},

		symbolizationProxyFallbackCount: s.reg.Counter("symbolization.proxy_fallback.count"),
	}
}

type RunConfig struct {
	MetricsPort uint32
	HTTPPort    uint32
	GRPCPort    uint32
}

func (s *PerforatorServer) runMetricsServer(ctx context.Context, port uint32) error {
	s.l.Info(ctx, "Starting metrics server", log.UInt32("port", port))
	mux := http.NewServeMux()
	mux.Handle("/metrics", s.reg.HTTPHandler(ctx, s.l))
	mux.HandleFunc("GET /debug/pprof/polyheap", polyheapprof.ServeCurrentHeapProfile)

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
		BaseContext: func(listener net.Listener) context.Context {
			return ctx
		},
	}

	unreg := context.AfterFunc(ctx, func() {
		s.l.Info(ctx, "Stopping metrics server")
		err := srv.Close()
		if err != nil {
			s.l.Error(ctx, "Failed to shutdown metrics server", log.Error(err))
		} else {
			s.l.Info(ctx, "Metrics server shutdown complete")
		}
	})
	defer unreg()

	return srv.ListenAndServe()
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

	unreg := context.AfterFunc(ctx, func() {
		s.l.Info(ctx, "Stopping GRPC server", log.Error(context.Cause(ctx)))
		s.grpcServer.Stop()
		s.l.Info(ctx, "GRPC server shutdown complete")
	})
	defer unreg()

	return s.grpcServer.Serve(lis)
}

func (s *PerforatorServer) runHTTPServer(ctx context.Context, port uint32) error {
	s.l.Info(ctx, "Starting HTTP REST server on port", log.UInt32("port", port))
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: s.httpRouter,
		BaseContext: func(listener net.Listener) context.Context {
			return ctx
		},
	}

	unreg := context.AfterFunc(ctx, func() {
		s.l.Info(ctx, "Stopping HTTP server")
		err := srv.Close()
		if err != nil {
			s.l.Error(ctx, "HTTP server shutdown failed", log.Error(err))
		} else {
			s.l.Info(ctx, "HTTP server shutdown complete")
		}
	})
	defer unreg()

	return srv.ListenAndServe()
}

func (s *PerforatorServer) Run(ctx context.Context, conf *RunConfig) error {
	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		err := s.downloader.RunBackgroundDownloader(ctx)
		if err != nil {
			s.l.Error(ctx, "Failed background downloader", log.Error(err))
		}
		return err
	})

	g.Go(func() error {
		err := s.bannedUsers.RunPoller(ctx)
		if err != nil {
			s.l.Error(ctx, "Banned users poller failed", log.Error(err))
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

	if s.bpClient != nil {
		g.Go(func() error {
			s.l.Info(ctx, "Starting binary processor client")
			err := s.bpClient.Run(ctx)
			if err != nil {
				s.l.Error(ctx, "Binary processor client failed", log.Error(err))
			}

			return err
		})
	}

	g.Go(func() error {
		err := s.runAsyncTasks(ctx)
		if err != nil {
			s.l.Error(ctx, "Async tasks runner failed", log.Error(err))
		}
		return err
	})

	err := g.Wait()
	if err != nil {
		return fmt.Errorf("proxy server failed: %w", err)
	}
	return nil
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
		return nil, status.Errorf(codes.InvalidArgument, "invalid profile: %v", err)
	}

	var metadata *meta.ProfileMetadata
	metadata, err = makeUploadProfileMeta(ctx, req.GetProfileMeta(), profile)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid metadata: %v", err)
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
			return nil, status.Errorf(codes.InvalidArgument, "invalid profile metadata: no event type found")
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
