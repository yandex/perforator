package service

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/google/pprof/profile"
	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/types/known/timestamppb"

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
	"github.com/yandex/perforator/perforator/proto/lib/time_interval"
	proto "github.com/yandex/perforator/perforator/proto/perforator"
	"github.com/yandex/perforator/perforator/symbolizer/pkg/client"
)

const (
	orderByProfiles = "profiles"
	uiTaskPath      = "/task/"
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

	// Defines the base URL prefix used to generate task URLs (e.g., "my.perforator/task/<taskID>")
	uiURLPrefix string

	metrics *MonitoringServiceMetrics
}

func NewMonitoringService(
	cfg *config.Config,
	logger log.Logger,
	reg xmetrics.Registry,
) (service *MonitoringService, err error) {
	ctx := context.Background()

	host, _, err := net.SplitHostPort(cfg.Client.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to get perforator host: %w", err)
	}
	// For example my.perforator/task/
	uiURLPrefix := host + uiTaskPath

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
		cfg:         cfg,
		cli:         cli,
		reg:         reg,
		uiURLPrefix: uiURLPrefix,
	}
	service.registerMetrics()

	return service, nil
}

////////////////////////////////////////////////////////////////////////////////

func serviceToSelectorService(service string) (string, error) {
	return profilequerylang.SelectorToString(profilequerylang.NewBuilder().Services(service).Build())
}

func (s *MonitoringService) gatherServicesMetrics(ctx context.Context, logger log.Logger, format *client.RenderFormat) error {
	listServicesCtx, cancel := context.WithTimeout(ctx, s.cfg.Timeout)
	defer cancel()
	services, err := s.cli.Client().ListServices(listServicesCtx, s.cfg.ServicesOffset, s.cfg.ServicesNumberToCheck, nil, nil, orderByProfiles)
	if err != nil {
		logger.Error("failed to list services", log.Error(err))
		return err
	}

	logger.Debug("Number of services", log.Int("number of services", len(services)))

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
					logger.Error("failed to gather metrics", log.Error(err), log.String("service id", service.ServiceID))
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

// taskIDtoUIURL returns url that directs to UI with corrisponding task
// For example my.perforator/task/<taskID>
func (s *MonitoringService) taskIDtoUIURL(taskID string) string {
	return s.uiURLPrefix + taskID
}

// This function makes merge profiles request for a service in some time interval and gathers metrics such as
// max stack depth, average frames number and unsymbolised locations number.
func (s *MonitoringService) gatherServiceProfilesMetrics(ctx context.Context, logger log.Logger, service string, format *client.RenderFormat, maxSamples uint32) error {
	ctx, cancel := context.WithTimeout(ctx, s.cfg.Timeout)
	defer cancel()
	logger = log.With(logger, log.String("service id", service))

	ToTS := time.Now()
	FromTS := ToTS.Add(-s.cfg.CheckQualityInterval)

	builder := profilequerylang.NewBuilder().
		From(FromTS).
		To(ToTS).
		Services(service)

	selector, err := profilequerylang.SelectorToString(builder.Build())
	if err != nil {
		logger.Error("Failed to create selector for service", log.Error(err))
		return err
	}

	start := time.Now()
	logger.Info("Merging profile")

	taskId, res, err := s.cli.Client().MergeProfilesProto(
		ctx,
		&proto.MergeProfilesRequest{
			Query: &proto.ProfileQuery{
				Selector: selector,
				TimeInterval: &time_interval.TimeInterval{
					From: timestamppb.New(FromTS),
					To:   timestamppb.New(ToTS),
				},
			},
			MaxSamples: maxSamples,
			Format:     format,
		},
	)

	if err != nil {
		s.metrics.mergeProfilesRequests.fails.Inc()
		s.reg.WithTags(Tags{"user_service": service, "status": "fail"}).Counter("requests.merge_profiles").Inc()
		logger.Error("failed to merge Profiles", log.Error(err), log.String("selector", selector))
		return err
	}

	logger.Info("Downloading profile")
	data, err := s.cli.Client().GetProfileByURL(res.GetProfileURL())
	if err != nil {
		s.metrics.mergeProfilesRequests.fails.Inc()
		s.reg.WithTags(Tags{"user_service": service, "status": "fail"}).Counter("requests.merge_profiles").Inc()
		logger.Error("failed to download Profile", log.Error(err), log.String("selector", selector), log.String("url", res.GetProfileURL()))
		return err
	}

	s.metrics.mergeProfilesRequests.successes.Inc()
	s.reg.WithTags(Tags{"user_service": service, "status": "success"}).Counter("requests.merge_profiles").Inc()

	s.metrics.mergeProfilesTimer.RecordDuration(time.Since(start))
	s.reg.WithTags(Tags{"user_service": service}).Timer("profile.merge").RecordDuration(time.Since(start))

	if len(res.ProfileMeta) == 0 {
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
		logger.Error("failed to parse profile", log.Error(err), log.String("selector", selector))
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

	logfields := []log.Field{
		log.String("url", s.taskIDtoUIURL(taskId)),
		log.String("unsymbolized percent", fmt.Sprintf("%.2f", float64(accum.UnsymbolizedNumberSum())/float64(accum.StackFramesSum())*100)),
		log.String("selector", selector),
	}

	if accum.UnsymbolizedNumberSum() == 0 {
		logger.Info("Succesfully parsed profile", logfields...)
	} else {
		logger.Warn("Parsed profile has unsymbolized locations", logfields...)
	}

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
			logger.Error("failed to gather services metrics", log.Error(err))
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
			logger.Error("failed metrics server", log.Error(err))
		}
		return err
	})

	g.Go(func() error {
		err := s.runProfileChecker(ctx, logger)
		if err != nil {
			logger.Error("profile checker stoped with error", log.Error(err))
		}
		return err
	})

	return g.Wait()
}
