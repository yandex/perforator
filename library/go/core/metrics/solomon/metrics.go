package solomon

import (
	"bytes"
	"context"
	"encoding"
	"encoding/json"
	"fmt"
	"io"
	"slices"
	"time"

	"github.com/yandex/perforator/library/go/core/xerrors"
)

// Gather collects all metrics data via snapshots.
func (r Registry) Gather() (*Metrics, error) {
	metrics := make([]Metric, 0)

	var err error
	r.metrics.Range(func(_, v any) bool {
		if s, ok := v.(Metric); ok {
			metrics = append(metrics, s.Snapshot())
			return true
		}
		err = fmt.Errorf("unexpected value type: %T", v)
		return false
	})

	if err != nil {
		return nil, err
	}

	return &Metrics{
		metrics:         metrics,
		commonStartTime: r.startTime,
	}, nil
}

func NewMetrics(metrics []Metric, opts ...MetricOpt) Metrics {
	var mopts MetricsOpts
	for _, opt := range opts {
		opt(&mopts)
	}

	return Metrics{
		metrics:         metrics,
		timestamp:       mopts.timestamp,
		commonLabels:    mopts.commonLabels,
		commonStartTime: mopts.commonStartTime,
	}
}

type valueType uint8

const (
	valueTypeNone         valueType = iota
	valueTypeOneWithoutTS valueType = 0x01
	valueTypeOneWithTS    valueType = 0x02
	valueTypeManyWithTS   valueType = 0x03
)

type metricType uint8

const (
	typeUnspecified    metricType = iota
	typeGauge          metricType = 0x01
	typeCounter        metricType = 0x02
	typeRated          metricType = 0x03
	typeIGauge         metricType = 0x04
	typeHistogram      metricType = 0x05
	typeRatedHistogram metricType = 0x06
)

func (k metricType) String() string {
	switch k {
	case typeCounter:
		return "COUNTER"
	case typeGauge:
		return "DGAUGE"
	case typeIGauge:
		return "IGAUGE"
	case typeHistogram:
		return "HIST"
	case typeRated:
		return "RATE"
	case typeRatedHistogram:
		return "HIST_RATE"
	default:
		panic("unknown metric type")
	}
}

// Metric is an any abstract solomon Metric.
type Metric interface {
	json.Marshaler

	// Returns the metric name.
	Name() string

	// Returns metric labels.
	// The returned value is for reading only. Do not try to modify it.
	Labels() map[string]string

	// Returns the metric value.
	// The returned value is for reading only. Do not try to modify it.
	Value() any

	// Snapshot returns independent copy on metric.
	Snapshot() Metric

	getType() metricType
	getNameTag() string
	getTimestamp() *time.Time
	getStartTime() uint32
	isMemOnly() bool
	getID() string

	writeSpackValue(w io.Writer) error
}

// Rated marks given Solomon metric or vector as rated.
// Example:
//
//	cnt := r.Counter("mycounter")
//	Rated(cnt)
//
//	cntvec := r.CounterVec("mycounter", []string{"mytag"})
//	Rated(cntvec)
//
// For additional info: https://m.yandex-team.ru/docs/concepts/data-model#rate
func Rated(s any) any {
	switch st := s.(type) {
	case *Counter:
		st.metricType = typeRated
	case *FuncCounter:
		st.metricType = typeRated
	case *Histogram:
		st.metricType = typeRatedHistogram

	case *CounterVec:
		st.vec.rated = true
	case *HistogramVec:
		st.vec.rated = true
	case *DurationHistogramVec:
		st.vec.rated = true
	}
	// All other metric types cannot be rated
	return s
}

// MemOnly marks given Solomon metric as mem-only.
func MemOnly(m any) any {
	if iface, ok := m.(interface{ setMemOnly() }); ok {
		iface.setMemOnly()
	}
	return m
}

var (
	_ json.Marshaler           = (*Metrics)(nil)
	_ encoding.BinaryMarshaler = (*Metrics)(nil)
)

type Metrics struct {
	metrics         []Metric
	timestamp       *time.Time
	commonLabels    map[string]string
	commonStartTime uint32
}

// MarshalJSON implements json.Marshaler.
func (s Metrics) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Metrics      []Metric          `json:"metrics"`
		Timestamp    *int64            `json:"ts,omitempty"`
		CommonLabels map[string]string `json:"commonLabels,omitempty"`
	}{s.metrics, tsAsRef(s.timestamp), s.commonLabels})
}

// MarshalBinary implements encoding.BinaryMarshaler.
func (s Metrics) MarshalBinary() ([]byte, error) {
	var buf bytes.Buffer
	se := NewSpackEncoder(context.Background(), CompressionNone, &s)
	n, err := se.Encode(&buf)
	if err != nil {
		return nil, xerrors.Errorf("encode only %d bytes: %w", n, err)
	}
	return buf.Bytes(), nil
}

// SplitToChunks splits Metrics into a slice of chunks, each at most maxChunkSize long.
// The length of returned slice is always at least one.
// Zero maxChunkSize denotes unlimited chunk length.
func (s Metrics) SplitToChunks(maxChunkSize int) []Metrics {
	if maxChunkSize == 0 || len(s.metrics) == 0 {
		return []Metrics{s}
	}

	chunks := make([]Metrics, 0, len(s.metrics)/maxChunkSize+1)
	for chunk := range slices.Chunk(s.metrics, maxChunkSize) {
		chunks = append(chunks, Metrics{
			metrics:         chunk,
			timestamp:       s.timestamp,
			commonLabels:    s.commonLabels,
			commonStartTime: s.commonStartTime,
		})
	}
	return chunks
}

func (s *Metrics) SetTimestamp(timestamp time.Time) {
	s.timestamp = &timestamp
}

func (s *Metrics) SetCommonStartTime(startTime time.Time) {
	s.commonStartTime = uint32(startTime.Unix())
}

func (s *Metrics) SetCommonLabels(labels map[string]string) {
	s.commonLabels = labels
}

func (s *Metrics) CommonLabels() map[string]string {
	return s.commonLabels
}

// List return list of metrics
func (s Metrics) List() []Metric {
	return slices.Clone(s.metrics)
}

func (s Metrics) Len() int {
	return len(s.metrics)
}

func tsAsRef(t *time.Time) *int64 {
	if t == nil {
		return nil
	}
	ts := t.Unix()
	return &ts
}
