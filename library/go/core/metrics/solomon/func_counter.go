package solomon

import (
	"io"

	"go.uber.org/atomic"
)

var _ Metric = (*FuncCounter)(nil)

// FuncCounter tracks int64 value returned by function.
type FuncCounter struct {
	baseMetric

	function func() int64
}

func NewFuncCounter(name string, function func() int64, opts ...MetricOpt) FuncCounter {
	mOpts := MetricsOpts{}
	for _, op := range opts {
		op(&mOpts)
	}
	mType := typeCounter
	if mOpts.rated {
		mType = typeRated
	}
	return FuncCounter{
		baseMetric: newBaseMetric(name, mType, opts...),
		function:   function,
	}
}

func (c *FuncCounter) Function() func() int64 {
	return c.function
}

func (c *FuncCounter) Value() any {
	return c.function()
}

func (c *FuncCounter) writeSpackValue(w io.Writer) error {
	return writeUint64LE(w, uint64(c.function()))
}

// MarshalJSON implements json.Marshaler.
func (c *FuncCounter) MarshalJSON() ([]byte, error) {
	return marshalScalarMetric(&c.baseMetric, c.function())
}

// Snapshot returns independent copy on metric.
func (c *FuncCounter) Snapshot() Metric {
	return &Counter{
		baseMetric: c.copy(),
		value:      *atomic.NewInt64(c.function()),
	}
}
