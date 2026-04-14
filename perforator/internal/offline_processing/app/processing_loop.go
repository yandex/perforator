package app

import (
	"context"
	"database/sql"
	"errors"
	"sync/atomic"
	"time"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/library/go/core/metrics"
	"github.com/yandex/perforator/library/go/ptr"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

type processingMetrics struct {
	success metrics.Counter
	failure metrics.Counter
	lag     metrics.Gauge
}

type ProcessingLoop struct {
	l       xlog.Logger
	metrics processingMetrics

	binarySelector BinarySelector
	binaryFetcher  BinaryFetcher
	processors     []BinaryProcessor

	lastIterationCompletion atomic.Pointer[time.Time]
}

func NewProcessingLoop(
	l xlog.Logger,
	reg metrics.Registry,

	binarySelector BinarySelector,
	binaryFetcher BinaryFetcher,
	processors []BinaryProcessor,
) (*ProcessingLoop, error) {
	loop := &ProcessingLoop{
		l: l,
		metrics: processingMetrics{
			success: reg.Counter("success"),
			failure: reg.Counter("failure"),
			lag:     reg.Gauge("processing_lag_seconds"),
		},
		binarySelector: binarySelector,
		binaryFetcher:  binaryFetcher,
		processors:     processors,
	}
	loop.lastIterationCompletion.Store(ptr.T(time.Now()))
	reg.FuncGauge("time_without_progress", func() float64 {
		delay := time.Since(*loop.lastIterationCompletion.Load())
		return delay.Seconds()
	})
	return loop, nil
}

func (l *ProcessingLoop) Run(ctx context.Context) error {
	l.runLoop(ctx)

	return nil
}

func (l *ProcessingLoop) runLoop(ctx context.Context) {
	for {
		err := l.loopIteration(ctx)

		l.processError(ctx, err)
	}
}

func (l *ProcessingLoop) loopIteration(ctx context.Context) error {
	// TODO : add some tracing spans here

	binaryHandler, err := l.binarySelector.SelectBinary(ctx)
	if err != nil {
		return err
	}
	defer func() {
		binaryHandler.Finalize(ctx, err)
	}()

	binary, err := l.binaryFetcher.FetchBinary(ctx, binaryHandler.GetBinaryID())
	if err != nil {
		return err
	}
	defer binary.Close()

	l.l.Info(ctx, "Started to process the binary", log.String("build_id", binaryHandler.GetBinaryID()))
	for _, processor := range l.processors {
		err = processor.ProcessBinary(
			ctx,
			binaryHandler,
			binaryHandler.GetBinaryID(),
			binary.Path(),
		)
		if err != nil {
			return err
		}
	}
	l.l.Info(ctx, "Successfully processed the binary", log.String("build_id", binaryHandler.GetBinaryID()))

	l.metrics.lag.Set(time.Since(binaryHandler.EnqueuedAt()).Seconds())
	l.lastIterationCompletion.Store(ptr.T(time.Now()))

	return nil
}

func (l *ProcessingLoop) processError(ctx context.Context, err error) {
	if err == nil {
		l.metrics.success.Add(1)
		return
	}

	// TODO : account for failures in metrics

	isQueueEmpty := errors.Is(err, sql.ErrNoRows)

	if isQueueEmpty {
		l.l.Info(ctx, "Queue is empty")
		// TODO : make this backoff more sophisticated
		time.Sleep(3 * time.Second)
		l.metrics.lag.Set(0)
		l.lastIterationCompletion.Store(ptr.T(time.Now()))
	} else {
		l.l.Warn(ctx, "Loop iteration failed", log.Error(err))
	}
}
