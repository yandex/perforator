package custom_profiling_operation

import (
	"context"
	"os"

	"golang.org/x/sync/errgroup"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/library/go/core/metrics"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/agent/custom_profiling_operation/models"
	"github.com/yandex/perforator/perforator/internal/agent_gateway/client/custom_profiling_operation"
	"github.com/yandex/perforator/perforator/pkg/xlog"
	cpo_proto "github.com/yandex/perforator/perforator/proto/custom_profiling_operation"
)

type ServiceConfig struct {
	PolledOperationsQueueSize uint64 `yaml:"polled_operations_queue_size"`
	Host                      string `yaml:"host"`
}

func (c *ServiceConfig) FillDefault() {
	if c.PolledOperationsQueueSize == 0 {
		c.PolledOperationsQueueSize = 100
	}
}

type serviceMetrics struct {
	failedHandlesCount metrics.Counter
	failedPollsCount   metrics.Counter
	successPollsCount  metrics.Counter
}

// Service is responsible for 2 things:
// 1) Polling custom profiling operations from the agent gateway
// 2) Handling custom profiling operations - enabling/disabling them, reporting their statuses
type Service struct {
	l                     xlog.Logger
	config                *ServiceConfig
	cpoClient             *custom_profiling_operation.Client
	polledOperationsQueue chan *cpo_proto.Operation
	handler               models.Handler
	metrics               serviceMetrics
}

func NewService(
	l xlog.Logger,
	reg metrics.Registry,
	config *ServiceConfig,
	cpoClient *custom_profiling_operation.Client,
	handler models.Handler,
) (*Service, error) {
	if config.Host == "" {
		var err error
		config.Host, err = os.Hostname()
		if err != nil {
			return nil, err
		}
	}

	reg = reg.WithPrefix("cpo_service")

	return &Service{
		l:                     l,
		config:                config,
		cpoClient:             cpoClient,
		polledOperationsQueue: make(chan *cpo_proto.Operation, config.PolledOperationsQueueSize),
		handler:               handler,
		metrics: serviceMetrics{
			failedHandlesCount: reg.WithTags(map[string]string{"status": "fail"}).Counter("handles.count"),
			successPollsCount:  reg.WithTags(map[string]string{"status": "success"}).Counter("polls.count"),
			failedPollsCount:   reg.WithTags(map[string]string{"status": "fail"}).Counter("polls.count"),
		},
	}, nil
}

func (p *Service) runPoller(ctx context.Context) error {
	defer close(p.polledOperationsQueue)

	poller := p.cpoClient.CreateLongPoller()
	for {
		// TODO: later add pod names argument. This requires moving deploy system to PerforatorAgent type
		operations, err := poller.PollOperations(ctx, p.config.Host, nil)
		if err != nil {
			p.metrics.failedPollsCount.Inc()
			p.l.Error(ctx, "Failed to poll operations", log.Error(err))
			continue
		}

		p.metrics.successPollsCount.Inc()
		p.l.Info(ctx, "Polled operations", log.Int("count", len(operations)))

		for _, operation := range operations {
			p.polledOperationsQueue <- operation
		}
	}
}

func (p *Service) runHandleLoop(ctx context.Context) error {
	for operation := range p.polledOperationsQueue {
		err := p.handler.Handle(ctx, operation)
		if err != nil {
			p.l.Error(ctx, "Failed to handle operation", log.Error(err))
			p.metrics.failedHandlesCount.Inc()
		}

		p.l.Info(ctx, "Handled operation", log.String("id", operation.ID), log.Any("spec", operation.Spec))
	}

	return nil
}

func (p *Service) Run(ctx context.Context) error {
	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		return p.runPoller(ctx)
	})

	g.Go(func() error {
		return p.runHandleLoop(ctx)
	})

	return g.Wait()
}
