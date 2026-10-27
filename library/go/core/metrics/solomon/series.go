package solomon

import (
	"encoding/binary"
	"encoding/json"
	"io"
	"math"
	"time"

	"github.com/yandex/perforator/library/go/core/xerrors"
)

type GaugePoint struct {
	Timestamp time.Time
	Value     float64
}

type CounterPoint struct {
	Timestamp time.Time
	Value     int64
}

type IGaugePoint struct {
	Timestamp time.Time
	Value     int64
}

const bytesPerSeriesPoint = 12 // ts uint32 + value uint64

func (p GaugePoint) encode(buf []byte) {
	binary.LittleEndian.PutUint32(buf, uint32(p.Timestamp.Unix()))
	binary.LittleEndian.PutUint64(buf[4:], math.Float64bits(p.Value))
}

func (p CounterPoint) encode(buf []byte) {
	binary.LittleEndian.PutUint32(buf, uint32(p.Timestamp.Unix()))
	binary.LittleEndian.PutUint64(buf[4:], uint64(p.Value))
}

func (p IGaugePoint) encode(buf []byte) {
	binary.LittleEndian.PutUint32(buf, uint32(p.Timestamp.Unix()))
	binary.LittleEndian.PutUint64(buf[4:], uint64(p.Value))
}

type pointEncoder interface {
	encode(buf []byte)
}

func packPoints[P pointEncoder](points []P) seriesData {
	if len(points) == 0 {
		return seriesData{}
	}
	buf := make([]byte, len(points)*bytesPerSeriesPoint)
	for i := range points {
		points[i].encode(buf[i*bytesPerSeriesPoint:])
	}
	return seriesData{
		pointCount: uint32(len(points)),
		packed:     buf,
	}
}

// seriesData holds pre-packed series points ready to be written into a spack stream.
type seriesData struct {
	pointCount uint32
	packed     []byte
}

func (s *seriesData) writeSpackValue(w io.Writer) error {
	if err := writeULEB128(w, s.pointCount); err != nil {
		return xerrors.Errorf("write series count: %w", err)
	}
	if _, err := w.Write(s.packed); err != nil {
		return xerrors.Errorf("write series points: %w", err)
	}
	return nil
}

func (s *seriesData) pointsCount() uint32 {
	return s.pointCount
}

// seriesMetric is satisfied by Metric implementations whose payload uses the
// valueTypeManyWithTS spack layout. The encoder dispatches on it.
type seriesMetric interface {
	Metric
	pointsCount() uint32
}

// GaugeSeries is an immutable batch of points for a single sensor.
// Timestamps are stored with second resolution.
type GaugeSeries struct {
	baseMetric
	seriesData
}

var (
	_ Metric       = (*GaugeSeries)(nil)
	_ seriesMetric = (*GaugeSeries)(nil)
)

// NewGaugeSeries packs the supplied points into a spack-ready blob.
func NewGaugeSeries(name string, points []GaugePoint, opts ...MetricOpt) GaugeSeries {
	return GaugeSeries{
		baseMetric: newBaseMetric(name, typeGauge, opts...),
		seriesData: packPoints(points),
	}
}

// Points unpacks the stored bytes into a fresh slice.
// Prefer not to call in hot code; the encoder works on the packed bytes directly.
func (s *GaugeSeries) Points() []GaugePoint {
	if s.pointCount == 0 {
		return nil
	}
	out := make([]GaugePoint, s.pointCount)
	for i := uint32(0); i < s.pointCount; i++ {
		off := i * bytesPerSeriesPoint
		ts := binary.LittleEndian.Uint32(s.packed[off:])
		v := math.Float64frombits(binary.LittleEndian.Uint64(s.packed[off+4:]))
		out[i] = GaugePoint{Timestamp: time.Unix(int64(ts), 0).UTC(), Value: v}
	}
	return out
}

func (s *GaugeSeries) Value() any {
	return s.Points()
}

func (s *GaugeSeries) Snapshot() Metric {
	return &GaugeSeries{
		baseMetric: s.copy(),
		seriesData: s.seriesData,
	}
}

func (s *GaugeSeries) MarshalJSON() ([]byte, error) {
	points := make([]seriesJSONPoint, s.pointCount)
	for i := uint32(0); i < s.pointCount; i++ {
		off := i * bytesPerSeriesPoint
		ts := binary.LittleEndian.Uint32(s.packed[off:])
		v := math.Float64frombits(binary.LittleEndian.Uint64(s.packed[off+4:]))
		points[i] = seriesJSONPoint{TS: int64(ts), Value: v}
	}
	return marshalSeriesJSON(&s.baseMetric, points)
}

// CounterSeries is an immutable batch of points for a counter sensor.
type CounterSeries struct {
	baseMetric
	seriesData
}

var (
	_ Metric       = (*CounterSeries)(nil)
	_ seriesMetric = (*CounterSeries)(nil)
)

// NewCounterSeries packs the supplied points; WithRated turns the series into a rate.
func NewCounterSeries(name string, points []CounterPoint, opts ...MetricOpt) CounterSeries {
	var mOpts MetricsOpts
	for _, op := range opts {
		op(&mOpts)
	}
	mType := typeCounter
	if mOpts.rated {
		mType = typeRated
	}
	return CounterSeries{
		baseMetric: newBaseMetric(name, mType, opts...),
		seriesData: packPoints(points),
	}
}

func (s *CounterSeries) Points() []CounterPoint {
	if s.pointCount == 0 {
		return nil
	}
	out := make([]CounterPoint, s.pointCount)
	for i := uint32(0); i < s.pointCount; i++ {
		off := i * bytesPerSeriesPoint
		ts := binary.LittleEndian.Uint32(s.packed[off:])
		v := int64(binary.LittleEndian.Uint64(s.packed[off+4:]))
		out[i] = CounterPoint{Timestamp: time.Unix(int64(ts), 0).UTC(), Value: v}
	}
	return out
}

func (s *CounterSeries) Value() any {
	return s.Points()
}

func (s *CounterSeries) Snapshot() Metric {
	return &CounterSeries{
		baseMetric: s.copy(),
		seriesData: s.seriesData,
	}
}

func (s *CounterSeries) MarshalJSON() ([]byte, error) {
	points := make([]seriesJSONPoint, s.pointCount)
	for i := uint32(0); i < s.pointCount; i++ {
		off := i * bytesPerSeriesPoint
		ts := binary.LittleEndian.Uint32(s.packed[off:])
		v := int64(binary.LittleEndian.Uint64(s.packed[off+4:]))
		points[i] = seriesJSONPoint{TS: int64(ts), Value: v}
	}
	return marshalSeriesJSON(&s.baseMetric, points)
}

// IGaugeSeries is an immutable batch of points for an integer gauge.
type IGaugeSeries struct {
	baseMetric
	seriesData
}

var (
	_ Metric       = (*IGaugeSeries)(nil)
	_ seriesMetric = (*IGaugeSeries)(nil)
)

func NewIGaugeSeries(name string, points []IGaugePoint, opts ...MetricOpt) IGaugeSeries {
	return IGaugeSeries{
		baseMetric: newBaseMetric(name, typeIGauge, opts...),
		seriesData: packPoints(points),
	}
}

func (s *IGaugeSeries) Points() []IGaugePoint {
	if s.pointCount == 0 {
		return nil
	}
	out := make([]IGaugePoint, s.pointCount)
	for i := uint32(0); i < s.pointCount; i++ {
		off := i * bytesPerSeriesPoint
		ts := binary.LittleEndian.Uint32(s.packed[off:])
		v := int64(binary.LittleEndian.Uint64(s.packed[off+4:]))
		out[i] = IGaugePoint{Timestamp: time.Unix(int64(ts), 0).UTC(), Value: v}
	}
	return out
}

func (s *IGaugeSeries) Value() any {
	return s.Points()
}

func (s *IGaugeSeries) Snapshot() Metric {
	return &IGaugeSeries{
		baseMetric: s.copy(),
		seriesData: s.seriesData,
	}
}

func (s *IGaugeSeries) MarshalJSON() ([]byte, error) {
	points := make([]seriesJSONPoint, s.pointCount)
	for i := uint32(0); i < s.pointCount; i++ {
		off := i * bytesPerSeriesPoint
		ts := binary.LittleEndian.Uint32(s.packed[off:])
		v := int64(binary.LittleEndian.Uint64(s.packed[off+4:]))
		points[i] = seriesJSONPoint{TS: int64(ts), Value: v}
	}
	return marshalSeriesJSON(&s.baseMetric, points)
}

type seriesJSONPoint struct {
	TS    int64 `json:"ts"`
	Value any   `json:"value"`
}

func marshalSeriesJSON(m *baseMetric, points []seriesJSONPoint) ([]byte, error) {
	return json.Marshal(struct {
		Type       string            `json:"type"`
		Labels     map[string]string `json:"labels"`
		Timeseries []seriesJSONPoint `json:"timeseries"`
		MemOnly    bool              `json:"memOnly,omitempty"`
	}{
		Type:       m.metricType.String(),
		Labels:     m.labelsWithName(),
		Timeseries: points,
		MemOnly:    m.memOnly,
	})
}
