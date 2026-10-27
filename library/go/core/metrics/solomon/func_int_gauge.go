package solomon

import (
	"io"

	"go.uber.org/atomic"
)

var _ Metric = (*FuncIntGauge)(nil)

// FuncIntGauge tracks int64 value returned by function.
type FuncIntGauge struct {
	baseMetric

	function func() int64
}

func NewFuncIntGauge(name string, function func() int64, opts ...MetricOpt) FuncIntGauge {
	return FuncIntGauge{
		baseMetric: newBaseMetric(name, typeIGauge, opts...),
		function:   function,
	}
}

func (g *FuncIntGauge) Function() func() int64 {
	return g.function
}

func (g *FuncIntGauge) Value() any {
	return g.function()
}

func (g *FuncIntGauge) writeSpackValue(w io.Writer) error {
	return writeUint64LE(w, uint64(g.function()))
}

// MarshalJSON implements json.Marshaler.
func (g *FuncIntGauge) MarshalJSON() ([]byte, error) {
	return marshalScalarMetric(&g.baseMetric, g.function())
}

// Snapshot returns independent copy on metric.
func (g *FuncIntGauge) Snapshot() Metric {
	return &IntGauge{
		baseMetric: g.copy(),
		value:      *atomic.NewInt64(g.function()),
	}
}
