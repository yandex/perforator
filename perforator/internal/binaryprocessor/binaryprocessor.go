package binaryprocessor

import (
	"context"
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
	"github.com/yandex/perforator/perforator/pkg/xlog"
	symbolizerproto "github.com/yandex/perforator/perforator/proto/symbolizer"
)

type BinaryProcessorServer struct {
	l   xlog.Logger
	c   *Config
	reg xmetrics.Registry

	symbolizer *symbolize.Symbolizer
	downloader *downloader.Downloader

	grpcServer   *grpc.Server
	healthServer *health.Server
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

	initCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	bgCtx := context.Background()
	storageBundle, err := bundle.NewStorageBundle(initCtx, bgCtx, l, "binproc", reg, &conf.StorageConfig)
	if err != nil {
		return nil, err
	}

	l.Info(ctx, "Initialized storage bundle")

	downloaderInstance, binaryDownloader, gsymDownloader, err := downloader.CreateDownloaders(
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
		downloader:   downloaderInstance,
		grpcServer:   grpcServer,
		healthServer: healthServer,
	}

	symbolizerproto.RegisterSymbolizerServer(server.grpcServer, server)
	reflection.Register(server.grpcServer)

	return server, nil
}

func (s *BinaryProcessorServer) Symbolize(ctx context.Context, r *symbolizerproto.SymbolizeRequest) (*symbolizerproto.SymbolizeResponse, error) {
	if r == nil {
		return nil, status.Error(codes.InvalidArgument, "nil request")
	}

	if r.Batch == nil {
		return nil, status.Error(codes.InvalidArgument, "nil batch")
	}

	for _, perBinaryRequest := range r.Batch {
		if perBinaryRequest == nil {
			return nil, status.Error(codes.InvalidArgument, "nil binary request")
		}
	}

	respBatch, err := s.symbolizer.SymbolizeBatch(ctx, r.Batch)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to symbolize: %v", err))
	}

	return &symbolizerproto.SymbolizeResponse{
		Batch: respBatch,
	}, nil
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

	err := g.Wait()
	if err != nil {
		return fmt.Errorf("binary processor server failed: %w", err)
	}
	return nil
}
