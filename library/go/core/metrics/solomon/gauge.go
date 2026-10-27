package solomon

import (
	"io"

	"go.uber.org/atomic"

	"github.com/yandex/perforator/library/go/core/metrics"
)

var (
	_ metrics.Gauge = (*Gauge)(nil)
	_ Metric        = (*Gauge)(nil)
)

// Gauge tracks single float64 value.
type Gauge struct {
	baseMetric

	value atomic.Float64
}

func NewGauge(name string, value float64, opts ...MetricOpt) Gauge {
	return Gauge{
		baseMetric: newBaseMetric(name, typeGauge, opts...),
		value:      *atomic.NewFloat64(value),
	}
}

func (g *Gauge) Set(value float64) {
	g.value.Store(value)
}

func (g *Gauge) Add(value float64) {
	g.value.Add(value)
}

func (g *Gauge) Value() any {
	return g.value.Load()
}

func (g *Gauge) writeSpackValue(w io.Writer) error {
	return writeFloat64LE(w, g.value.Load())
}

// MarshalJSON implements json.Marshaler.
func (g *Gauge) MarshalJSON() ([]byte, error) {
	return marshalScalarMetric(&g.baseMetric, g.value.Load())
}

// Snapshot returns independent copy of metric.
func (g *Gauge) Snapshot() Metric {
	return &Gauge{
		baseMetric: g.copy(),
		value:      *atomic.NewFloat64(g.value.Load()),
	}
}
