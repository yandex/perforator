package service

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/spf13/afero"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthgrpc "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/perforator/internal/xmetrics"
	"github.com/yandex/perforator/perforator/pkg/grpcutil/grpclog"
	"github.com/yandex/perforator/perforator/pkg/grpcutil/grpcmetrics"
	"github.com/yandex/perforator/perforator/pkg/polyheapprof"
	s3client "github.com/yandex/perforator/perforator/pkg/s3"
	"github.com/yandex/perforator/perforator/pkg/tracing"
	"github.com/yandex/perforator/perforator/pkg/xlog"
	"github.com/yandex/perforator/perforator/proto/perforator"
)

type WebService struct {
	l   xlog.Logger
	cfg *Config
	reg xmetrics.Registry

	otelShutdown func(context.Context) error
	httpRouter   http.Handler
	grpcServer   *grpc.Server
	healthServer *health.Server

	client *Client
}

func NewWebService(
	cfg *Config,
	l xlog.Logger,
	reg xmetrics.Registry,
	uiFS afero.Fs,
) (service *WebService, err error) {
	cfg.fillDefault()

	ctx := context.Background()

	// Setup OpenTelemetry tracing.
	exporter, err := tracing.NewExporter(ctx, cfg.Tracing)
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

	// Setup REST server.
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(middleware.Logger)
	//r.Use(otelhttp.NewMiddleware("http.server"))
	//r.Use(authp.HTTP().Middleware

	s3Handler, err := s3Handler(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to init s3 handler: %w", err)
	}

	indexHandler, err := indexHandler(uiFS)
	if err != nil {
		return nil, fmt.Errorf("failed to init index.html handler: %w", err)
	}

	apiHandler, err := apiHandler(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to init perforator proxy api handler: %w", err)
	}

	httpFs := afero.NewHttpFs(uiFS)
	fileServer := http.FileServer(httpFs.Dir("dist"))

	r.Handle("/assets/*", fileServer)
	r.Get("/static/results/{id}", s3Handler)
	r.Mount("/api", apiHandler)

	// Setup GRPC client.
	client, err := NewClient(cfg.ClientConfig, l)

	// Setup GRPC server.
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
			metricsInterceptor.UnaryServer(),
			logInterceptor.UnaryServer(),
		),
		grpc.ChainStreamInterceptor(
			metricsInterceptor.StreamServer(),
			logInterceptor.StreamServer(),
		),
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
	)

	healthServer := health.NewServer()
	healthgrpc.RegisterHealthServer(grpcServer, healthServer)

	service = &WebService{
		l:            l,
		cfg:          cfg,
		reg:          reg,
		otelShutdown: shutdown,
		httpRouter:   wrapNotFound(r, http.HandlerFunc(indexHandler)),
		grpcServer:   grpcServer,
		healthServer: healthServer,
		client:       client,
	}

	perforator.RegisterPerforatorServer(service.grpcServer, service)
	perforator.RegisterTaskServiceServer(service.grpcServer, service)
	perforator.RegisterMicroscopeServiceServer(service.grpcServer, service)
	reflection.Register(service.grpcServer)

	return service, nil
}

// getFileFromS3AndWrite fetches a file from S3 and writes it to the ResponseWriter
func getFileFromS3AndWrite(w http.ResponseWriter, client *s3.S3, bucket, key string) error {
	input := &s3.GetObjectInput{
		Bucket: &bucket,
		Key:    &key,
	}

	resp, err := client.GetObjectWithContext(context.Background(), input)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	_, err = io.Copy(w, resp.Body)
	return err
}

func s3Handler(cfg *Config) (http.HandlerFunc, error) {
	s3Client, err := s3client.NewClient(cfg.S3Config)
	if err != nil {
		return nil, fmt.Errorf("failed to init s3: %w", err)
	}

	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		err := getFileFromS3AndWrite(w, s3Client, cfg.RenderedProfilesStorageConfig.S3Bucket, id)
		if err != nil {
			//TODO: add logging
			http.Error(w, "File not found", http.StatusNotFound)
			return
		}
	}, nil

}

func apiHandler(cfg *Config) (http.Handler, error) {
	targetURL, err := url.Parse(fmt.Sprintf("http://%s/", cfg.ClientConfig.HTTPHost))
	if err != nil {
		return nil, err
	}
	proxy := httputil.NewSingleHostReverseProxy(targetURL)

	return proxy, nil
}

func indexHandler(uiFS afero.Fs) (http.HandlerFunc, error) {
	f, err := uiFS.Open("dist/index.html")
	if err != nil {
		return nil, err
	}
	defer f.Close()

	index, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}

	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		//TODO: add logging.
		_, _ = w.Write(index)
	}, nil

}

func (s *WebService) runMetricsServer(ctx context.Context, port uint) error {
	s.l.Info(ctx, "Starting metrics server", log.UInt("port", port))
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

func (s *WebService) runHTTPServer(ctx context.Context, port uint) error {
	s.l.Info(ctx, "Starting HTTP REST server on port", log.UInt("port", port))
	srv := &http.Server{Addr: fmt.Sprintf(":%d", port), Handler: s.httpRouter}
	return srv.ListenAndServe()
}

func (s *WebService) runGRPCServer(ctx context.Context, port uint) error {
	s.l.Info(ctx, "Starting GRPC server", log.UInt("port", port))

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return err
	}

	s.healthServer.SetServingStatus("", healthgrpc.HealthCheckResponse_SERVING)
	return s.grpcServer.Serve(lis)
}

func (s *WebService) Run(ctx context.Context, conf *PortsConfig) error {
	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		err := s.runMetricsServer(ctx, conf.MetricsPort)
		if err != nil {
			s.l.Error(ctx, "Failed metrics server", log.Error(err))
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
		err := s.runGRPCServer(ctx, conf.GRPCPort)
		if err != nil {
			s.l.Error(ctx, "GRPC server failed", log.Error(err))
		}
		return err
	})

	return g.Wait()
}
