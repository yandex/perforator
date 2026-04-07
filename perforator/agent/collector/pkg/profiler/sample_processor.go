package profiler

import (
	"context"
	"errors"
	"fmt"
	"math"
	"runtime"
	"sync/atomic"

	"golang.org/x/sys/unix"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/library/go/core/metrics"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/config"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/machine"
	"github.com/yandex/perforator/perforator/internal/unwinder"
	"github.com/yandex/perforator/perforator/pkg/graceful"
	"github.com/yandex/perforator/perforator/pkg/pubsub"
)

type sampleUnmarshaller struct {
	sample *unwinder.RecordSample
	bypass bool
}

func (u *sampleUnmarshaller) UnmarshalBinary(data []byte) error {
	if u.bypass {
		return u.sample.UnmarshalBinaryUnsafe(data)
	}

	return u.sample.UnmarshalBinary(data)
}

type GlobalProcessedWatermark = uint64

func NewGlobalProcessedWatermark(monotonicTime unix.Timespec) GlobalProcessedWatermark {
	return GlobalProcessedWatermark(uint64(monotonicTime.Sec)*1_000_000_000 + uint64(monotonicTime.Nsec))
}

// sampleProcessor owns the perfbuf sample reader loop and the sample consumer
// registry. It publishes each sample's collection_time via PubSub so that
// callers can implement barrier-style waiting.
type sampleProcessor struct {
	log  log.Logger
	bpf  *machine.BPF
	conf *config.SampleConsumerConfig

	sampleCallback      machine.RawSampleCallback
	bypassSampleParsing bool

	consumers SampleConsumerRegistry

	shutdownCookie graceful.ShutdownCookie

	collectionTimePubSub *pubsub.PubSub[GlobalProcessedWatermark]
	cpuWatermarks        []atomic.Uint64
	lastGlobalWatermark  GlobalProcessedWatermark

	samplesDuration metrics.Counter
}

type sampleProcessorOptions struct {
	bypassSampleParsing bool
}

type sampleProcessorOption func(*sampleProcessorOptions)

func WithSampleParsingBypass(bypass bool) sampleProcessorOption {
	return func(o *sampleProcessorOptions) {
		o.bypassSampleParsing = bypass
	}
}

func newSampleProcessor(
	l log.Logger,
	r metrics.Registry,
	bpf *machine.BPF,
	conf *config.SampleConsumerConfig,
	sampleCallback machine.RawSampleCallback,
	consumers SampleConsumerRegistry,
	opts ...sampleProcessorOption,
) *sampleProcessor {
	options := &sampleProcessorOptions{}
	for _, opt := range opts {
		opt(options)
	}

	return &sampleProcessor{
		log:                  l,
		bpf:                  bpf,
		conf:                 conf,
		sampleCallback:       sampleCallback,
		bypassSampleParsing:  options.bypassSampleParsing,
		consumers:            consumers,
		shutdownCookie:       graceful.NewShutdownCookie(),
		collectionTimePubSub: pubsub.NewPubSub[GlobalProcessedWatermark](pubsub.WithNonBlockingPublish()),
		cpuWatermarks:        make([]atomic.Uint64, runtime.NumCPU()),
		samplesDuration:      r.Counter("sample_duration.nsec"),
	}
}

// WaitForSampleProcessing blocks.
// Postcondition:
// If S is a call to WaitForSampleProcessing, then every sample collection
// which starts before S starts, finishes before S returns.
//
// Note: This relies on the assumption that events are being generated across
// all logical CPUs. If some CPU does not generate samples, the global watermark will not
// advance, and this method will get stuck
func (sp *sampleProcessor) waitForSampleProcessing(ctx context.Context) error {
	var ts unix.Timespec
	if err := unix.ClockGettime(unix.CLOCK_MONOTONIC, &ts); err != nil {
		return fmt.Errorf("failed to get CLOCK_MONOTONIC: %w", err)
	}
	threshold := NewGlobalProcessedWatermark(ts)

	const reasonableSubscriberChannelCapacity = 10
	sub := sp.collectionTimePubSub.Subscribe(pubsub.WithChanCapacity(reasonableSubscriberChannelCapacity))
	defer sub.Close()

	for {
		select {
		case globalWatermark, ok := <-sub.Chan():
			if !ok {
				return errors.New("sample processor stopped, channel closed")
			}
			if globalWatermark > threshold {
				return nil
			}
		case <-ctx.Done():
			return context.Cause(ctx)
		}
	}
}

// GracefulStop signals the sample reader to drain remaining samples and stop.
func (sp *sampleProcessor) GracefulStop(ctx context.Context) error {
	return sp.shutdownCookie.Stop(ctx)
}

// Run starts the sample reading loop. It blocks until the context is cancelled
// or a graceful shutdown is requested.
func (sp *sampleProcessor) Run(ctx context.Context) error {
	stopSource := sp.shutdownCookie.GetSource()
	defer stopSource.Finish()
	defer sp.collectionTimePubSub.CloseAll()

	reader, err := sp.openSampleReader(*sp.conf.PerfBufferWatermark, sp.sampleCallback)
	if err != nil {
		return err
	}
	defer reader.Close()

	unmarshaller := &sampleUnmarshaller{
		sample: &unwinder.RecordSample{},
		bypass: sp.bypassSampleParsing,
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-stopSource.Done():
			goto gracefulstop
		default:
		}

		sp.readSample(ctx, reader, unmarshaller)
	}

gracefulstop:
	sp.log.Debug("Graceful shutdown has been requested, going to drain sample queue")

	for sp.readSample(ctx, reader, unmarshaller) {
	}

	sp.log.Debug("Restarting sample reader in order to consume last non-notified samples")
	_ = reader.Close()
	reader, err = sp.openSampleReader(0, nil)
	if err != nil {
		return err
	}

	for sp.readSample(ctx, reader, unmarshaller) {
	}

	return nil
}

func (sp *sampleProcessor) openSampleReader(watermark int, sampleCallback machine.RawSampleCallback) (*machine.PerfReader, error) {
	opts := &machine.PerfReaderOptions{
		PerCPUBufferSize: *sp.conf.PerfBufferPerCPUSize,
		Watermark:        watermark,
		SampleCallback:   sampleCallback,
	}
	return sp.bpf.MakeSampleReader(opts)
}

func (sp *sampleProcessor) readSample(ctx context.Context, reader *machine.PerfReader, unmarshaller *sampleUnmarshaller) bool {
	meta, err := reader.Read(ctx, unmarshaller)
	if err != nil {
		return false
	}

	sample := unmarshaller.sample
	sp.samplesDuration.Add(int64(sample.Runtime))

	consumers := sp.consumers.Consumers()
	for _, consumer := range consumers {
		consumer.Consume(ctx, sample)
	}

	if meta.CPU >= 0 && meta.CPU < len(sp.cpuWatermarks) {
		sp.cpuWatermarks[meta.CPU].Store(sample.CollectionTime)
	}

	minTime := GlobalProcessedWatermark(math.MaxUint64)
	for i := range sp.cpuWatermarks {
		t := GlobalProcessedWatermark(sp.cpuWatermarks[i].Load())
		if t < minTime {
			minTime = t
		}
	}
	if minTime != math.MaxUint64 && minTime > sp.lastGlobalWatermark {
		sp.lastGlobalWatermark = minTime
		sp.collectionTimePubSub.Publish(minTime)
	}

	return true
}
