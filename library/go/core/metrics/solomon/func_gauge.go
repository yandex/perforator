package solomon

import (
	"io"

	"go.uber.org/atomic"
)

var _ Metric = (*FuncGauge)(nil)

// FuncGauge tracks float64 value returned by function.
type FuncGauge struct {
	baseMetric

	function func() float64
}

func NewFuncGauge(name string, function func() float64, opts ...MetricOpt) FuncGauge {
	return FuncGauge{
		baseMetric: newBaseMetric(name, typeGauge, opts...),
		function:   function,
	}
}

func (g *FuncGauge) Function() func() float64 {
	return g.function
}

func (g *FuncGauge) Value() any {
	return g.function()
}

func (g *FuncGauge) writeSpackValue(w io.Writer) error {
	return writeFloat64LE(w, g.function())
}

// MarshalJSON implements json.Marshaler.
func (g *FuncGauge) MarshalJSON() ([]byte, error) {
	return marshalScalarMetric(&g.baseMetric, g.function())
}

// Snapshot returns independent copy on metric.
func (g *FuncGauge) Snapshot() Metric {
	return &Gauge{
		baseMetric: g.copy(),
		value:      *atomic.NewFloat64(g.function()),
	}
}
