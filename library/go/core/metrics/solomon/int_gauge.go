package solomon

import (
	"encoding/json"
	"io"

	"go.uber.org/atomic"

	"github.com/yandex/perforator/library/go/core/metrics"
)

var (
	_ metrics.IntGauge = (*IntGauge)(nil)
	_ Metric           = (*IntGauge)(nil)
)

// IntGauge tracks single int64 value.
type IntGauge struct {
	baseMetric

	value atomic.Int64
}

func NewIntGauge(name string, value int64, opts ...MetricOpt) IntGauge {
	return IntGauge{
		baseMetric: newBaseMetric(name, typeIGauge, opts...),
		value:      *atomic.NewInt64(value),
	}
}

func (g *IntGauge) Set(value int64) {
	g.value.Store(value)
}

func (g *IntGauge) Add(value int64) {
	g.value.Add(value)
}

func (g *IntGauge) Value() any {
	return g.value.Load()
}

func (g *IntGauge) writeSpackValue(w io.Writer) error {
	return writeUint64LE(w, uint64(g.value.Load()))
}

// MarshalJSON implements json.Marshaler.
func (g *IntGauge) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Type      string            `json:"type"`
		Labels    map[string]string `json:"labels"`
		Value     int64             `json:"value"`
		Timestamp *int64            `json:"ts,omitempty"`
		MemOnly   bool              `json:"memOnly,omitempty"`
	}{
		Type:      g.metricType.String(),
		Value:     g.value.Load(),
		Labels:    g.labelsWithName(),
		Timestamp: tsAsRef(g.timestamp),
		MemOnly:   g.memOnly,
	})
}

// Snapshot returns independent copy of metric.
func (g *IntGauge) Snapshot() Metric {
	return &IntGauge{
		baseMetric: g.copy(),
		value:      *atomic.NewInt64(g.value.Load()),
	}
}
