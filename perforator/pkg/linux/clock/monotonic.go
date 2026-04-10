package clock

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"golang.org/x/sys/unix"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/library/go/core/metrics"
)

// MonotonicTime represents nanoseconds elapsed since system boot (excluding suspend time).
// It corresponds to the value returned by BPF helper bpf_ktime_get_ns().
type MonotonicTime = uint64

// MonotonicClockConverter is responsible for converting MonotonicTime to wall-clock time.Time.
type MonotonicClockConverter struct {
	delta atomic.Int64
	l     log.Logger
	r     metrics.Registry
}

func NewMonotonicClockConverter(l log.Logger, r metrics.Registry) (*MonotonicClockConverter, error) {
	c := &MonotonicClockConverter{
		l: l,
		r: r,
	}

	if err := c.updateDelta(); err != nil {
		return nil, fmt.Errorf("failed to initialize time delta: %w", err)
	}

	return c, nil
}

// Run updates the delta between Monotonic and Realtime clocks once a second.
// This is necessary to compensate for time adjustments (NTP adjtime) and time jumps.
func (c *MonotonicClockConverter) Run(ctx context.Context) error {
	errorCounter := c.r.Counter("clock.delta_update_errors.count")
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := c.updateDelta(); err != nil {
				errorCounter.Inc()
				c.l.Error("Failed to update clock delta", log.Error(err))
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (c *MonotonicClockConverter) updateDelta() error {
	var realtime, monotonic unix.Timespec

	if err := unix.ClockGettime(unix.CLOCK_REALTIME, &realtime); err != nil {
		return err
	}
	if err := unix.ClockGettime(unix.CLOCK_MONOTONIC, &monotonic); err != nil {
		return err
	}

	const nanosecondsPerSecond = 1_000_000_000
	realtimeNanos := realtime.Sec*nanosecondsPerSecond + realtime.Nsec
	monotonicNanos := monotonic.Sec*nanosecondsPerSecond + monotonic.Nsec

	delta := realtimeNanos - monotonicNanos

	c.delta.Store(delta)
	return nil
}

func (c *MonotonicClockConverter) MonotonicToTime(m MonotonicTime) time.Time {
	delta := c.delta.Load()
	wallTimeNanos := int64(m) + delta

	return time.Unix(0, wallTimeNanos)
}
