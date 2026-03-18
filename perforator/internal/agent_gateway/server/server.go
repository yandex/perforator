package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/perforator/internal/agent_gateway/server/custom_profiling_operation"
	"github.com/yandex/perforator/perforator/internal/agent_gateway/server/storage"
	"github.com/yandex/perforator/perforator/internal/xmetrics"
	"github.com/yandex/perforator/perforator/pkg/grpcutil/grpcmetrics"
	"github.com/yandex/perforator/perforator/pkg/storage/bundle"
	"github.com/yandex/perforator/perforator/pkg/storage/creds"
	storagetvm "github.com/yandex/perforator/perforator/pkg/storage/tvm"
	"github.com/yandex/perforator/perforator/pkg/xlog"
	custom_profiling_operation_proto "github.com/yandex/perforator/perforator/proto/custom_profiling_operation"
	perforatorstorage "github.com/yandex/perforator/perforator/proto/storage"
)

type Server struct {
	storageService                  *storage.Service
	customProfilingOperationService *custom_profiling_operation.Service

	logger xlog.Logger
	reg    xmetrics.Registry
	conf   *Config

	grpcServer *grpc.Server
}

func (s *Server) runGrpcServer(ctx context.Context) error {
	s.logger.Info(ctx, "Starting profile storage server", log.UInt32("port", s.conf.Port))

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", s.conf.Port))
	if err != nil {
		return err
	}

	unreg := context.AfterFunc(ctx, func() {
		s.logger.Info(ctx, "Stopping Agent Gateway GRPC server")
		s.grpcServer.Stop()
		s.logger.Info(ctx, "Agent Gateway GRPC server shutdown complete")
	})
	defer unreg()

	if err = s.grpcServer.Serve(lis); err != nil {
		s.logger.Error(ctx, "Failed to grpc server", log.Error(err))
	}

	return err
}

func (s *Server) runMetricsServer(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.Handle("/metrics", s.reg.HTTPHandler(ctx, s.logger))
	port := s.conf.MetricsPort
	if port == 0 {
		port = 85
	}
	s.logger.Info(ctx, "Starting Agent Gateway metrics server", log.UInt32("port", port))

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}

	unreg := context.AfterFunc(ctx, func() {
		s.logger.Info(ctx, "Stopping Agent Gateway metrics server")
		_ = srv.Close()
	})
	defer unreg()

	return srv.ListenAndServe()
}

func (s *Server) Run(ctx context.Context) error {
	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		return s.runGrpcServer(ctx)
	})

	g.Go(func() error {
		return s.storageService.Run(ctx)
	})

	if s.customProfilingOperationService != nil {
		g.Go(func() error {
			return s.customProfilingOperationService.Run(ctx)
		})
	}

	g.Go(func() error {
		return s.runMetricsServer(ctx)
	})

	return g.Wait()
}

func NewServer(
	conf *Config,
	logger xlog.Logger,
	registry xmetrics.Registry,
	storageServiceOptions ...storage.Option,
) (*Server, error) {
	if err := ValidateConfig(conf); err != nil {
		return nil, fmt.Errorf("failed to validate config: %w", err)
	}
	conf.FillDefault()

	initCtx, cancel := context.WithTimeout(context.TODO(), 5*time.Second)
	defer cancel()

	// TODO: this context should be tied to e.g. Run() duration.
	bgCtx := context.TODO()

	storageBundle, err := bundle.NewStorageBundle(initCtx, bgCtx, logger, "agent_gateway", registry, &conf.StorageConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create storage bundle: %w", err)
	}

	storageServiceConfig := conf.StorageServiceConfig
	if storageServiceConfig == nil {
		storageServiceConfig = conf.StorageServiceConfigDeprecated
	}

	storageService, err := storage.NewService(
		storageServiceConfig,
		logger,
		registry,
		storageBundle,
		storageServiceOptions...,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create storage service: %w", err)
	}

	var customProfilingOperationService *custom_profiling_operation.Service
	if conf.CustomProfilingOperationServiceConfig != nil {
		customProfilingOperationService, err = custom_profiling_operation.NewService(
			logger,
			registry,
			conf.CustomProfilingOperationServiceConfig,
			storageBundle.CustomProfilingOperationStorage,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create custom profiling operation service: %w", err)
		}
	}

	server := &Server{
		logger:                          logger,
		conf:                            conf,
		reg:                             registry,
		storageService:                  storageService,
		customProfilingOperationService: customProfilingOperationService,
	}

	var grpcOpts []grpc.ServerOption

	tlsOpts, err := conf.TLS.GRPCServerOptions()
	if err != nil {
		return nil, fmt.Errorf("failed to configure TLS: %w", err)
	}
	grpcOpts = append(grpcOpts, tlsOpts...)

	var unaryServerInterceptors []grpc.UnaryServerInterceptor
	var streamServerInterceptors []grpc.StreamServerInterceptor

	credsInterceptor, err := getInterceptor(conf, logger)
	if err != nil {
		return nil, err
	}
	if credsInterceptor != nil {
		unaryServerInterceptors = append(unaryServerInterceptors, credsInterceptor.Unary())
		streamServerInterceptors = append(streamServerInterceptors, credsInterceptor.Stream())
	}

	metricsInterceptor := grpcmetrics.NewMetricsInterceptor(registry)
	unaryServerInterceptors = append(unaryServerInterceptors, metricsInterceptor.UnaryServer())
	streamServerInterceptors = append(streamServerInterceptors, metricsInterceptor.StreamServer())

	grpcOpts = append(grpcOpts, grpc.ChainUnaryInterceptor(unaryServerInterceptors...))
	grpcOpts = append(grpcOpts, grpc.ChainStreamInterceptor(streamServerInterceptors...))

	grpcOpts = append(grpcOpts, grpc.MaxRecvMsgSize(1024*1024*1024 /* 1 GB */))

	server.grpcServer = grpc.NewServer(grpcOpts...)
	perforatorstorage.RegisterPerforatorStorageServer(server.grpcServer, server.storageService)
	if server.customProfilingOperationService != nil {
		custom_profiling_operation_proto.RegisterCustomProfilingOperationServiceServer(server.grpcServer, server.customProfilingOperationService)
	}
	reflection.Register(server.grpcServer)

	return server, nil
}

func getInterceptor(conf *Config, logger xlog.Logger) (creds.ServerInterceptor, error) {
	if conf.TvmAuth != nil {
		return storagetvm.NewTVMServerInterceptor(
			conf.TvmAuth.ID,
			os.Getenv(conf.TvmAuth.SecretEnvName),
			conf.TvmAuth.AllowedIDs,
			logger,
		)
	}
	return nil, nil
}
