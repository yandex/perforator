package solomon

import (
	"io"

	"go.uber.org/atomic"

	"github.com/yandex/perforator/library/go/core/metrics"
)

var (
	_ metrics.Counter = (*Counter)(nil)
	_ Metric          = (*Counter)(nil)
)

// Counter tracks monotonically increasing value.
type Counter struct {
	baseMetric

	value atomic.Int64
}

func NewCounter(name string, value int64, opts ...MetricOpt) Counter {
	mOpts := MetricsOpts{}
	for _, op := range opts {
		op(&mOpts)
	}

	mType := typeCounter
	if mOpts.rated {
		mType = typeRated
	}

	return Counter{
		baseMetric: newBaseMetric(name, mType, opts...),
		value:      *atomic.NewInt64(value),
	}
}

// Inc increments counter by 1.
func (c *Counter) Inc() {
	c.Add(1)
}

// Add adds delta to the counter. Delta must be >=0.
func (c *Counter) Add(delta int64) {
	c.value.Add(delta)
}

func (c *Counter) Value() any {
	return c.value.Load()
}

func (c *Counter) writeSpackValue(w io.Writer) error {
	return writeUint64LE(w, uint64(c.value.Load()))
}

// MarshalJSON implements json.Marshaler.
func (c *Counter) MarshalJSON() ([]byte, error) {
	return marshalScalarMetric(&c.baseMetric, c.value.Load())
}

// Snapshot returns independent copy on metric.
func (c *Counter) Snapshot() Metric {
	return &Counter{
		baseMetric: c.copy(),
		value:      *atomic.NewInt64(c.value.Load()),
	}
}
