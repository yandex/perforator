package service

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/google/pprof/profile"
	"golang.org/x/sync/errgroup"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/library/go/core/metrics"
	"github.com/yandex/perforator/library/go/ptr"
	"github.com/yandex/perforator/perforator/internal/symbolizer/cli"
	"github.com/yandex/perforator/perforator/internal/symbolizer/pprofmetrics"
	"github.com/yandex/perforator/perforator/internal/symbolizer/quality_monitoring/internal/config"
	"github.com/yandex/perforator/perforator/internal/xmetrics"
	"github.com/yandex/perforator/perforator/pkg/profilequerylang"
	"github.com/yandex/perforator/perforator/pkg/tracing"
	"github.com/yandex/perforator/perforator/pkg/xlog"
	proto "github.com/yandex/perforator/perforator/proto/perforator"
	"github.com/yandex/perforator/perforator/symbolizer/pkg/client"
)

const (
	orderByProfiles = "profiles"
)

type Tags = map[string]string

type requestsMetrics struct {
	successes metrics.Counter
	fails     metrics.Counter
}

type MonitoringServiceMetrics struct {
	stackMaxDepth             metrics.Counter
	stackAverageFramesNumber  metrics.Counter
	samplesNumber             metrics.Counter
	unsymbolizedAverageNumber metrics.Counter
	profilesCounter           metrics.Counter

	mergeProfilesRequests requestsMetrics
	mergeProfilesTimer    metrics.Timer
}

func (s *MonitoringService) registerMetrics() {
	s.metrics = &MonitoringServiceMetrics{
		stackMaxDepth:             s.reg.WithTags(Tags{"user_service": "all"}).Counter("stack.max_depth"),
		stackAverageFramesNumber:  s.reg.WithTags(Tags{"user_service": "all"}).Counter("frames.count"),
		samplesNumber:             s.reg.WithTags(Tags{"user_service": "all"}).Counter("samples.count"),
		unsymbolizedAverageNumber: s.reg.WithTags(Tags{"user_service": "all"}).Counter("frames.unsymbolized.count"),
		profilesCounter:           s.reg.WithTags(Tags{"user_service": "all"}).Counter("profiles.count"),
		mergeProfilesTimer:        s.reg.WithTags(Tags{"user_service": "all"}).Timer("profile.merge"),
		mergeProfilesRequests: requestsMetrics{
			successes: s.reg.WithTags(Tags{"user_service": "all", "status": "success"}).Counter("requests.merge_profiles"),
			fails:     s.reg.WithTags(Tags{"user_service": "all", "status": "fail"}).Counter("requests.merge_profiles"),
		},
	}
}

type MonitoringService struct {
	cfg *config.Config
	reg xmetrics.Registry
	cli *cli.App

	metrics *MonitoringServiceMetrics
}

func NewMonitoringService(
	cfg *config.Config,
	logger log.Logger,
	reg xmetrics.Registry,
) (service *MonitoringService, err error) {
	ctx := context.Background()

	// Setup OpenTelemetry tracing.
	exporter, err := tracing.NewExporter(ctx, cfg.Tracing)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize tracing span exporter: %w", err)
	}

	shutdown, _, err := tracing.Initialize(ctx, logger.WithName("tracing"), exporter, "perforator", "monitoring")
	if err != nil {
		return nil, fmt.Errorf("failed to initialize tracing: %w", err)
	}
	defer func() {
		if err != nil && shutdown != nil {
			_ = shutdown(ctx)
		}
	}()
	logger.Info("Successfully initialized tracing")

	cli, err := cli.New(&cli.Config{Client: &cfg.Client})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize perforator CLI: %s", err)
	}
	logger.Info("Created perforator CLI")

	service = &MonitoringService{
		cfg: cfg,
		cli: cli,
		reg: reg,
	}
	service.registerMetrics()

	return service, nil
}

////////////////////////////////////////////////////////////////////////////////

func serviceToSelectorService(service string) (string, error) {
	return profilequerylang.SelectorToString(profilequerylang.NewBuilder().Services(service).Build())
}

func (s *MonitoringService) gatherServicesMetrics(ctx context.Context, logger log.Logger, format *client.RenderFormat) error {
	services, err := s.cli.Client().ListServices(ctx, s.cfg.ServicesOffset, s.cfg.ServicesNumberToCheck, nil, nil, orderByProfiles)
	if err != nil {
		logger.Error("Failed to list services", log.Error(err))
		return err
	}

	logger.Debug("number of services", log.Int("number of services", len(services)))

	var wg sync.WaitGroup
	servicesCh := make(chan *proto.ServiceMeta)

	for i := 0; i < s.cfg.ServicesCheckingConcurrency; i++ {
		wg.Add(1)
		go func(servicesCh <-chan *proto.ServiceMeta, wg *sync.WaitGroup) {
			defer wg.Done()

			for service := range servicesCh {
				logger.Info("Gathering metrics", log.String("service id", service.ServiceID))
				err := s.gatherServiceProfilesMetrics(ctx, logger, service.ServiceID, format, s.cfg.MaxSamplesToMerge)
				if err != nil {
					logger.Error("Failed to gather metrics", log.Error(err), log.String("service id", service.ServiceID))
					continue
				}
			}
		}(servicesCh, &wg)
	}

	for _, service := range services {
		servicesCh <- service
	}
	close(servicesCh)

	wg.Wait()
	logger.Info("Finisned current iteration", log.Time("time", time.Now()))

	return nil
}

// This function makes merge profiles request for a service in some time interval and gathers metrics such as
// max stack depth, average frames number and unsymbolised locations number.
func (s *MonitoringService) gatherServiceProfilesMetrics(ctx context.Context, logger log.Logger, service string, format *client.RenderFormat, maxSamples uint32) error {
	logger = log.With(logger, log.String("service id", service))

	ToTS := time.Now()
	FromTS := ToTS.Add(-s.cfg.CheckQualityInterval)

	serviceSelector, err := serviceToSelectorService(service)
	if err != nil {
		logger.Error("Failed to create selector for service", log.Error(err))
		return err
	}

	start := time.Now()
	logger.Info("Fetching profile")
	data, metas, err := s.cli.Client().MergeProfiles(
		ctx,
		&client.MergeProfilesRequest{
			ProfileFilters: client.ProfileFilters{
				Selector: serviceSelector,
				FromTS:   FromTS,
				ToTS:     ToTS,
			},
			MaxSamples: maxSamples,
			Format:     format,
		},
		false,
	)
	if err != nil {
		s.metrics.mergeProfilesRequests.fails.Inc()
		s.reg.WithTags(Tags{"user_service": service, "status": "fail"}).Counter("requests.merge_profiles").Inc()
		logger.Error("Failed to merge Profiles", log.Error(err))
		return err
	}
	s.metrics.mergeProfilesRequests.successes.Inc()
	s.reg.WithTags(Tags{"user_service": service, "status": "success"}).Counter("requests.merge_profiles").Inc()

	s.metrics.mergeProfilesTimer.RecordDuration(time.Since(start))
	s.reg.WithTags(Tags{"user_service": service}).Timer("profile.merge").RecordDuration(time.Since(start))

	if len(metas) == 0 {
		logger.Warn("There are no profiles to merge")
		return nil
	}

	if len(data) == 0 {
		logger.Warn("Merged profile is empty")
		return nil
	}

	logger.Info("Parsing profile")
	p, err := profile.Parse(bytes.NewBuffer(data))
	if err != nil {
		//TODO: add meta info
		logger.Error("Failed to parse profile", log.Error(err))
		return err
	}
	accum := pprofmetrics.NewProfileMetricsAccumulator(p)

	// Add metrics for each service separately.
	s.reg.WithTags(Tags{"user_service": service}).Counter("stack.max_depth").Add(accum.StackMaxDepth())
	s.reg.WithTags(Tags{"user_service": service}).Counter("frames.count").Add(accum.StackFramesSum())
	s.reg.WithTags(Tags{"user_service": service}).Counter("samples.count").Add(accum.SamplesNumber())
	s.reg.WithTags(Tags{"user_service": service}).Counter("frames.unsymbolized.count").Add(accum.UnsymbolizedNumberSum())
	s.reg.WithTags(Tags{"user_service": service}).Counter("profiles.count").Inc()

	// Add metrics to total count.
	s.metrics.stackMaxDepth.Add(accum.StackMaxDepth())
	s.metrics.stackAverageFramesNumber.Add(accum.StackFramesSum())
	s.metrics.samplesNumber.Add(accum.SamplesNumber())
	s.metrics.unsymbolizedAverageNumber.Add(accum.UnsymbolizedNumberSum())
	s.metrics.profilesCounter.Inc()

	return nil
}

////////////////////////////////////////////////////////////////////////////////

type RunConfig struct {
	MetricsPort uint
}

func (s *MonitoringService) runMetricsServer(ctx context.Context, logger log.Logger, port uint) error {
	logger.Infof("Starting metrics server on port %d", port)
	http.Handle("/metrics", s.reg.HTTPHandler(ctx, xlog.New(logger)))
	return http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
}

func (s *MonitoringService) runProfileChecker(ctx context.Context, logger log.Logger) error {
	defer s.Shutdown()
	logger = logger.WithName("ProfileChecker")

	var (
		format = &proto.RenderFormat{
			Symbolize: &proto.SymbolizeOptions{
				Symbolize: ptr.Bool(true),
			},
			Format: &proto.RenderFormat_RawProfile{
				RawProfile: &proto.RawProfileOptions{},
			},
		}
	)

	ticker := time.NewTicker(s.cfg.IterationSplay)
	defer ticker.Stop()

	logger.Info("Entering the loop")
	for {
		//TODO: add human readable time
		logger.Info("Starting a new iteration", log.Time("time", time.Now()))
		err := s.gatherServicesMetrics(ctx, logger, format)
		if err != nil {
			logger.Error("Failed to gather services metrics", log.Error(err))
			logger.Info("Finisned current iteration", log.Time("time", time.Now()))
			time.Sleep(s.cfg.SleepAfterFailedServicesChecking)
			continue
		}
		logger.Info("Finisned current iteration", log.Time("time", time.Now()))

		select {
		case <-ctx.Done():
			logger.Info("Exiting the loop")
			return ctx.Err()
		case <-ticker.C:
			continue
		}
	}
}

func (s *MonitoringService) Shutdown() {
	s.cli.Shutdown()
}

func (s *MonitoringService) Run(ctx context.Context, logger log.Logger, conf *RunConfig) error {
	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		err := s.runMetricsServer(ctx, logger, conf.MetricsPort)
		if err != nil {
			logger.Error("Failed metrics server", log.Error(err))
		}
		return err
	})

	g.Go(func() error {
		err := s.runProfileChecker(ctx, logger)
		if err != nil {
			logger.Error("Profile checker stoped with error", log.Error(err))
		}
		return err
	})

	return g.Wait()
}
