package signal_profile_processor

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/library/go/core/metrics"
	"github.com/yandex/perforator/perforator/internal/xmetrics"
	"github.com/yandex/perforator/perforator/pkg/kafka/consumer"
	"github.com/yandex/perforator/perforator/pkg/kafka/producer"
	"github.com/yandex/perforator/perforator/pkg/profile_event"
	"github.com/yandex/perforator/perforator/pkg/xlog"
	"github.com/yandex/perforator/perforator/symbolizer/pkg/client"
)

type Service struct {
	logger xlog.Logger
	cfg    EventProcessorConfig
	reg    xmetrics.Registry
	proc   Processor

	signalProfileEventConsumer *consumer.JSONConsumer[profile_event.SignalProfileEvent]
	CoreEventProducer          *producer.JSONProducer[profile_event.CoreEventMessage]

	whitelist   map[string]struct{}
	messageChan chan *profile_event.SignalProfileEvent
	metrics     *serviceMetrics
}

type serviceMetrics struct {
	messagesCountConsumed      metrics.Counter
	messagesCountFiltered      metrics.Counter
	messagesCountProcessedOK   metrics.Counter
	messagesCountProcessedFail metrics.Counter
	messagesCountPublishedOK   metrics.Counter
	messagesCountPublishedFail metrics.Counter
	consumeErrors              metrics.Counter

	messageQueueLength metrics.Gauge

	processTimerOK   metrics.Timer
	processTimerFail metrics.Timer
	publishTimerOK   metrics.Timer
	publishTimerFail metrics.Timer
}

func newServiceMetrics(reg xmetrics.Registry) *serviceMetrics {
	return &serviceMetrics{
		messagesCountConsumed:      reg.WithTags(map[string]string{"kind": "consumed"}).Counter("messages.count"),
		messagesCountFiltered:      reg.WithTags(map[string]string{"kind": "filtered"}).Counter("messages.count"),
		messagesCountProcessedOK:   reg.WithTags(map[string]string{"kind": "processed_success"}).Counter("messages.count"),
		messagesCountProcessedFail: reg.WithTags(map[string]string{"kind": "processed_fail"}).Counter("messages.count"),
		messagesCountPublishedOK:   reg.WithTags(map[string]string{"kind": "published_success"}).Counter("messages.count"),
		messagesCountPublishedFail: reg.WithTags(map[string]string{"kind": "published_fail"}).Counter("messages.count"),
		consumeErrors:              reg.Counter("consume_errors.count"),

		messageQueueLength: reg.Gauge("message_queue_length.gauge"),

		processTimerOK:   reg.WithTags(map[string]string{"kind": "success"}).Timer("process.timer"),
		processTimerFail: reg.WithTags(map[string]string{"kind": "fail"}).Timer("process.timer"),
		publishTimerOK:   reg.WithTags(map[string]string{"kind": "success"}).Timer("publish.timer"),
		publishTimerFail: reg.WithTags(map[string]string{"kind": "fail"}).Timer("publish.timer"),
	}
}

// NewService creates a new event processor service.
func NewService(log xlog.Logger, cfg Config, reg xmetrics.Registry) (*Service, error) {
	if cfg.KafkaConsumer == nil {
		return nil, errors.New("kafka_consumer config is required")
	}
	if cfg.KafkaProducer == nil {
		return nil, errors.New("kafka_producer config is required")
	}

	client, err := client.NewClient(context.Background(), &cfg.ProxyClient, log.WithName("client"))
	if err != nil {
		return nil, fmt.Errorf("failed to initialize client: %w", err)
	}
	proc := NewProxyProcessor(client)
	kc, err := consumer.NewKafkaConsumer(log, cfg.KafkaConsumer)
	if err != nil {
		return nil, fmt.Errorf("build consumer: %w", err)
	}

	kp, err := producer.NewKafkaProducer(log, cfg.KafkaProducer)
	if err != nil {
		return nil, fmt.Errorf("build producer: %w", err)
	}

	svc, err := newEventProcessor(kp, kc, proc, log, cfg.EventProcessorConfig, reg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize event processor: %w", err)
	}

	return svc, nil
}

func newEventProcessor(p producer.Producer, c consumer.Consumer, proc Processor, log xlog.Logger, cfg EventProcessorConfig, reg xmetrics.Registry) (*Service, error) {
	cfg.fillDefaults()
	wl := make(map[string]struct{}, len(cfg.WhitelistServices))
	for _, s := range cfg.WhitelistServices {
		if s == "" {
			continue
		}
		wl[s] = struct{}{}
	}

	return &Service{
		logger:                     log.WithName("signal_profile_processor"),
		cfg:                        cfg,
		reg:                        reg,
		proc:                       proc,
		signalProfileEventConsumer: consumer.NewJSONConsumer[profile_event.SignalProfileEvent](c),
		CoreEventProducer:          producer.NewJSONProducer[profile_event.CoreEventMessage](p),
		whitelist:                  wl,
		messageChan:                make(chan *profile_event.SignalProfileEvent, cfg.QueueSize),
		metrics:                    newServiceMetrics(reg),
	}, nil
}

// Run starts the consume process produce loop.
func (s *Service) Run(ctx context.Context) error {
	defer func() {
		if err := s.CoreEventProducer.Close(); err != nil {
			s.logger.Error(ctx, "Failed to close producer", log.Error(err))
		}
		if err := s.signalProfileEventConsumer.Close(); err != nil {
			s.logger.Error(ctx, "Failed to close consumer", log.Error(err))
		}
	}()

	go func() {
		s.consumeSignalProfileEvents(ctx)
		close(s.messageChan)
	}()

	var wg sync.WaitGroup
	for i := 0; i < s.cfg.WorkersNumber; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.processMessages(ctx)
		}()
	}
	wg.Wait()

	return ctx.Err()
}

// consumeSignalProfileEvents consumes signal profile events, applies whitelist before enqueue, then sends event to the channel.
func (s *Service) consumeSignalProfileEvents(ctx context.Context) {
	for {
		in, err := s.signalProfileEventConsumer.Consume(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return
			}
			s.metrics.consumeErrors.Inc()
			s.logger.Error(ctx, "Failed to consume event", log.Error(err))
			//TODO: maybe add sleep to prevent transient errors log spam.
			continue
		}
		s.metrics.messagesCountConsumed.Inc()

		if len(s.whitelist) > 0 {
			if _, ok := s.whitelist[in.Service]; !ok {
				s.metrics.messagesCountFiltered.Inc()
				s.logger.Info(ctx, "Filtered by whitelist",
					log.String("service", in.Service),
					log.String("profile_id", in.ProfileID),
				)
				continue
			}
		}

		select {
		case s.messageChan <- in:
			s.metrics.messageQueueLength.Set(float64(len(s.messageChan)))
		case <-ctx.Done():
			return
		}
	}
}

// processMessages runs a worker loop.
func (s *Service) processMessages(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case in, ok := <-s.messageChan:
			if !ok {
				return
			}
			s.metrics.messageQueueLength.Set(float64(len(s.messageChan)))
			procStart := time.Now()
			out, err := s.proc.Process(ctx, in)
			if err != nil {
				if errors.Is(err, context.Canceled) {
					return
				}
				s.metrics.messagesCountProcessedFail.Inc()
				s.metrics.processTimerFail.RecordDuration(time.Since(procStart))

				s.logger.Error(ctx, "Failed to process event",
					log.Error(err),
					log.String("service", in.Service),
					log.String("profile_id", in.ProfileID),
				)
				continue
			}
			s.metrics.messagesCountProcessedOK.Inc()
			s.metrics.processTimerOK.RecordDuration(time.Since(procStart))

			pubStart := time.Now()
			headers := []producer.Header{
				{Key: profile_event.CoreEventServiceKey, Value: []byte(in.Service)},
			}
			if err := s.CoreEventProducer.Publish(ctx, []byte(out.PartitionKey), out.Event, headers...); err != nil {
				if errors.Is(err, context.Canceled) {
					return
				}
				s.metrics.messagesCountPublishedFail.Inc()
				s.metrics.publishTimerFail.RecordDuration(time.Since(pubStart))

				s.logger.Error(ctx, "Failed to publish core event",
					log.Error(err),
					log.String("key", out.PartitionKey),
					log.String("service", out.Event.Core.Service),
				)
				continue
			}
			s.metrics.messagesCountPublishedOK.Inc()
			s.metrics.publishTimerOK.RecordDuration(time.Since(pubStart))
		}
	}
}
