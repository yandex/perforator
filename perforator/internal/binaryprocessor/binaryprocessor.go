package binaryprocessor

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health"
	healthgrpc "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/perforator/internal/symbolizer/binaryprovider/downloader"
	"github.com/yandex/perforator/perforator/internal/symbolizer/symbolize"
	"github.com/yandex/perforator/perforator/internal/xmetrics"
	"github.com/yandex/perforator/perforator/pkg/grpcutil/grpclog"
	"github.com/yandex/perforator/perforator/pkg/grpcutil/grpcmetrics"
	"github.com/yandex/perforator/perforator/pkg/polyheapprof"
	"github.com/yandex/perforator/perforator/pkg/storage/bundle"
	"github.com/yandex/perforator/perforator/pkg/tracing"
	"github.com/yandex/perforator/perforator/pkg/xlog"
	symbolizerproto "github.com/yandex/perforator/perforator/proto/symbolizer"
)

type BinaryProcessorServer struct {
	l   xlog.Logger
	c   *Config
	reg xmetrics.Registry

	symbolizer *symbolize.Symbolizer

	grpcServer   *grpc.Server
	healthServer *health.Server

	shutdownTracing func(context.Context) error
}

func getSymbolizationMode(conf *Config) symbolize.SymbolizationMode {
	if conf.SymbolizationConfig.UseGSYM {
		return symbolize.SymbolizationModeGSYMPreferred
	} else {
		return symbolize.SymbolizationModeDWARF
	}
}

var _ symbolizerproto.SymbolizerServer = &BinaryProcessorServer{}

func NewBinaryProcessorServer(
	conf *Config,
	l xlog.Logger,
	reg xmetrics.Registry,
) (*BinaryProcessorServer, error) {
	ctx := context.Background()

	exporter, err := tracing.NewExporter(ctx, conf.Tracing)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize tracing span exporter: %w", err)
	}
	shutdownTracing, _, err := tracing.Initialize(ctx, l.WithName("tracing").Logger(), exporter, "perforator", "binproc")
	if err != nil {
		return nil, fmt.Errorf("failed to initialize tracing: %w", err)
	}
	l.Info(ctx, "Successfully initialized tracing")

	initCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	bgCtx := context.TODO()
	storageBundle, err := bundle.NewStorageBundle(initCtx, bgCtx, l, "binproc", reg, &conf.StorageConfig)
	if err != nil {
		return nil, err
	}

	l.Info(ctx, "Initialized storage bundle")

	binaryDownloader, gsymDownloader, err := downloader.CreateDownloaders(
		conf.BinaryProvider.FileCache,
		conf.BinaryProvider.MaxSimultaneousDownloads,
		l,
		reg,
		storageBundle.BinaryStorage, storageBundle.GSYMStorage,
	)

	if err != nil {
		return nil, err
	}

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

	logInterceptor := grpclog.
		NewLogInterceptor(l.WithName("grpc")).
		SkipMethods(healthgrpc.Health_Watch_FullMethodName).
		SkipMethods(healthgrpc.Health_Check_FullMethodName)

	metricsInterceptor := grpcmetrics.NewMetricsInterceptor(reg)

	grpcServer := grpc.NewServer(
		grpc.MaxSendMsgSize(1024*1024*1024 /*1G*/),
		grpc.MaxRecvMsgSize(1024*1024*1024 /*1G*/),
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
		grpc.ChainUnaryInterceptor(
			metricsInterceptor.UnaryServer(),
			logInterceptor.UnaryServer(),
		),
		grpc.ChainStreamInterceptor(
			otelgrpc.StreamServerInterceptor(),
			metricsInterceptor.StreamServer(),
			logInterceptor.StreamServer(),
		),
	)
	healthServer := health.NewServer()
	healthgrpc.RegisterHealthServer(grpcServer, healthServer)

	server := &BinaryProcessorServer{
		l:            l,
		c:            conf,
		reg:          reg,
		symbolizer:   symbolizer,
		grpcServer:   grpcServer,
		healthServer: healthServer,

		shutdownTracing: shutdownTracing,
	}

	symbolizerproto.RegisterSymbolizerServer(server.grpcServer, server)
	reflection.Register(server.grpcServer)

	return server, nil
}

func (s *BinaryProcessorServer) Symbolize(ctx context.Context, r *symbolizerproto.SymbolizeRequest) (*symbolizerproto.SymbolizeResponse, error) {
	if r.BuildID == "" {
		return nil, status.Error(codes.InvalidArgument, "empty build id")
	}

	addresses, err := s.symbolizer.Symbolize(ctx, r.BuildID, r.Addresses)
	if err != nil {
		if errors.Is(err, symbolize.ErrUnknownBinary) {
			return nil, status.Errorf(codes.NotFound, "binary %s not available: %v", r.BuildID, err)
		}
		if errors.Is(err, symbolize.ErrSymbolization) {
			return nil, status.Errorf(codes.FailedPrecondition, "binary %s could not be symbolized: %v", r.BuildID, err)
		}
		return nil, status.Errorf(codes.Internal, "failed to symbolize %s: %v", r.BuildID, err)
	}

	return &symbolizerproto.SymbolizeResponse{Addresses: addresses}, nil
}

type RunConfig struct {
	MetricsPort uint32
	GRPCPort    uint32
}

func (s *BinaryProcessorServer) runMetricsServer(ctx context.Context, port uint32) error {
	s.l.Info(ctx, "Starting metrics server", log.UInt32("port", port))
	http.Handle("/metrics", s.reg.HTTPHandler(ctx, s.l))
	http.HandleFunc("GET /debug/pprof/polyheap", polyheapprof.ServeCurrentHeapProfile)
	return http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
}

func (s *BinaryProcessorServer) runGRPCServer(ctx context.Context, port uint32) error {
	s.l.Info(ctx, "Starting binary processor server", log.UInt32("port", port))

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return err
	}

	s.healthServer.SetServingStatus("", healthgrpc.HealthCheckResponse_SERVING)
	s.healthServer.SetServingStatus("NPerforator.NProto.Symbolizer", healthgrpc.HealthCheckResponse_SERVING)

	return s.grpcServer.Serve(lis)
}

func (s *BinaryProcessorServer) Run(ctx context.Context, conf *RunConfig) error {
	defer func() { _ = s.shutdownTracing(context.Background()) }()

	g, ctx := errgroup.WithContext(ctx)

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

	err := g.Wait()
	if err != nil {
		return fmt.Errorf("binary processor server failed: %w", err)
	}
	return nil
}
