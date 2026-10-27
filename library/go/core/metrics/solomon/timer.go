package solomon

import (
	"io"
	"time"

	"go.uber.org/atomic"

	"github.com/yandex/perforator/library/go/core/metrics"
)

var (
	_ metrics.Timer = (*Timer)(nil)
	_ Metric        = (*Timer)(nil)
)

// Timer measures gauge duration.
type Timer struct {
	baseMetric

	value atomic.Duration
}

func NewTimer(name string, value time.Duration, opts ...MetricOpt) Timer {
	mOpts := MetricsOpts{}
	for _, op := range opts {
		op(&mOpts)
	}
	mType := typeGauge
	if mOpts.rated {
		mType = typeRated
	}
	return Timer{
		baseMetric: newBaseMetric(name, mType, opts...),
		value:      *atomic.NewDuration(value),
	}
}

func (t *Timer) RecordDuration(value time.Duration) {
	t.value.Store(value)
}

func (t *Timer) Value() any {
	return t.value.Load().Seconds()
}

func (t *Timer) writeSpackValue(w io.Writer) error {
	return writeFloat64LE(w, t.value.Load().Seconds())
}

// MarshalJSON implements json.Marshaler.
func (t *Timer) MarshalJSON() ([]byte, error) {
	return marshalScalarMetric(&t.baseMetric, t.value.Load().Seconds())
}

// Snapshot returns independent copy on metric.
func (t *Timer) Snapshot() Metric {
	return &Timer{
		baseMetric: t.copy(),
		value:      *atomic.NewDuration(t.value.Load()),
	}
}
