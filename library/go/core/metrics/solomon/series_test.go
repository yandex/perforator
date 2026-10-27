package solomon

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGaugeSeries_PointsRoundtrip(t *testing.T) {
	points := []GaugePoint{
		{Timestamp: time.Unix(1000, 0), Value: 1.5},
		{Timestamp: time.Unix(2000, 0), Value: -3.25},
		{Timestamp: time.Unix(3000, 0), Value: 0},
	}
	s := NewGaugeSeries("mygauge", points)

	require.Equal(t, uint32(3), s.pointsCount())
	got := s.Points()
	require.Len(t, got, 3)
	for i, p := range points {
		assert.Equal(t, p.Timestamp.Unix(), got[i].Timestamp.Unix())
		assert.Equal(t, p.Value, got[i].Value)
	}
}

func TestCounterSeries_PointsRoundtrip(t *testing.T) {
	points := []CounterPoint{
		{Timestamp: time.Unix(1000, 0), Value: 1},
		{Timestamp: time.Unix(2000, 0), Value: 1 << 40},
		{Timestamp: time.Unix(3000, 0), Value: -7},
	}
	s := NewCounterSeries("mycounter", points)

	require.Equal(t, uint32(3), s.pointsCount())
	got := s.Points()
	require.Len(t, got, 3)
	for i, p := range points {
		assert.Equal(t, p.Timestamp.Unix(), got[i].Timestamp.Unix())
		assert.Equal(t, p.Value, got[i].Value)
	}
	assert.Equal(t, typeCounter, s.getType())
}

func TestCounterSeries_Rated(t *testing.T) {
	s := NewCounterSeries("rated", []CounterPoint{{Timestamp: time.Unix(1, 0), Value: 1}}, WithRated(true))
	assert.Equal(t, typeRated, s.getType())
}

func TestIGaugeSeries_PointsRoundtrip(t *testing.T) {
	points := []IGaugePoint{
		{Timestamp: time.Unix(10, 0), Value: 42},
		{Timestamp: time.Unix(20, 0), Value: -42},
	}
	s := NewIGaugeSeries("myigauge", points)

	require.Equal(t, uint32(2), s.pointsCount())
	got := s.Points()
	require.Len(t, got, 2)
	for i, p := range points {
		assert.Equal(t, p.Timestamp.Unix(), got[i].Timestamp.Unix())
		assert.Equal(t, p.Value, got[i].Value)
	}
	assert.Equal(t, typeIGauge, s.getType())
}

func TestSeries_EmptyPoints(t *testing.T) {
	s := NewGaugeSeries("empty", nil)
	assert.Equal(t, uint32(0), s.pointsCount())
	assert.Nil(t, s.Points())

	var buf bytes.Buffer
	err := s.writeSpackValue(&buf)
	require.NoError(t, err)
	// ULEB128(0) is a single zero byte.
	assert.Equal(t, []byte{0x00}, buf.Bytes())
}

func TestSeries_Snapshot(t *testing.T) {
	s := NewGaugeSeries("g", []GaugePoint{{Timestamp: time.Unix(1, 0), Value: 1}})
	snap := s.Snapshot()
	gs, ok := snap.(*GaugeSeries)
	require.True(t, ok)
	assert.Equal(t, s.Points(), gs.Points())
}

func TestSeries_MarshalJSON(t *testing.T) {
	s := NewGaugeSeries("mygauge", []GaugePoint{
		{Timestamp: time.Unix(1500000000, 0), Value: 1.5},
		{Timestamp: time.Unix(1500000060, 0), Value: 2.5},
	})

	data, err := s.MarshalJSON()
	require.NoError(t, err)

	var parsed struct {
		Type       string            `json:"type"`
		Labels     map[string]string `json:"labels"`
		Timeseries []struct {
			TS    int64   `json:"ts"`
			Value float64 `json:"value"`
		} `json:"timeseries"`
	}
	require.NoError(t, json.Unmarshal(data, &parsed))

	assert.Equal(t, "DGAUGE", parsed.Type)
	assert.Equal(t, "mygauge", parsed.Labels["sensor"])
	require.Len(t, parsed.Timeseries, 2)
	assert.Equal(t, int64(1500000000), parsed.Timeseries[0].TS)
	assert.Equal(t, 1.5, parsed.Timeseries[0].Value)
	assert.Equal(t, int64(1500000060), parsed.Timeseries[1].TS)
	assert.Equal(t, 2.5, parsed.Timeseries[1].Value)
}

// TestSeriesEncode verifies the exact spack byte layout for a GaugeSeries.
func TestSeriesEncode_GaugeSeries(t *testing.T) {
	s := NewGaugeSeries("mygauge", []GaugePoint{
		{Timestamp: time.Unix(1000, 0), Value: 1.5},
		{Timestamp: time.Unix(2000, 0), Value: 2.5},
	})

	metrics := &Metrics{metrics: []Metric{&s}}

	var buf bytes.Buffer
	enc := NewSpackEncoder(context.Background(), CompressionNone, metrics)
	written, err := enc.Encode(&buf)
	require.NoError(t, err)

	expectHeader := []byte{
		0x53, 0x50, // magic
		0x01, 0x01, // version 1.1
		0x18, 0x00, // header size = 24
		0x00,                   // time precision = SECONDS
		0x00,                   // compression = none
		0x07, 0x00, 0x00, 0x00, // label names size = 7 ("sensor\0")
		0x08, 0x00, 0x00, 0x00, // label values size = 8 ("mygauge\0")
		0x01, 0x00, 0x00, 0x00, // metric count = 1
		0x02, 0x00, 0x00, 0x00, // POINT COUNT = 2 (sum across series points)
	}

	expectPoolsAndCommon := []byte{
		// name pool
		's', 'e', 'n', 's', 'o', 'r', 0x00,
		// value pool
		'm', 'y', 'g', 'a', 'u', 'g', 'e', 0x00,
		// common time
		0x00, 0x00, 0x00, 0x00,
		// common labels count
		0x00,
	}

	expectMetric := []byte{
		0x07, // types: typeGauge(1) << 2 | valueTypeManyWithTS(3) = 0b00000111
		0x00, // flags
		0x01, // labels count
		0x00, // sensor name idx
		0x00, // mygauge value idx
		0x02, // point count = 2

		// point 1: ts=1000, value=1.5
		0xe8, 0x03, 0x00, 0x00, // 1000 LE
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xf8, 0x3f, // 1.5 LE (float64)

		// point 2: ts=2000, value=2.5
		0xd0, 0x07, 0x00, 0x00, // 2000 LE
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x04, 0x40, // 2.5 LE (float64)
	}

	body := buf.Bytes()
	require.True(t, bytes.HasPrefix(body, expectHeader), "header mismatch:\n got=%x\nwant=%x", body[:HeaderSize], expectHeader)
	body = body[HeaderSize:]

	require.True(t, bytes.HasPrefix(body, expectPoolsAndCommon), "pools/common mismatch:\n got=%x\nwant=%x", body[:len(expectPoolsAndCommon)], expectPoolsAndCommon)
	body = body[len(expectPoolsAndCommon):]

	require.Equal(t, expectMetric, body, "metric body mismatch")

	expectWritten := HeaderSize + len(expectPoolsAndCommon) + len(expectMetric)
	assert.Equal(t, expectWritten, written)
}

// TestSeriesEncode_CounterSeries verifies the byte layout for a CounterSeries.
func TestSeriesEncode_CounterSeries(t *testing.T) {
	s := NewCounterSeries("mycounter", []CounterPoint{
		{Timestamp: time.Unix(100, 0), Value: 42},
	})

	metrics := &Metrics{metrics: []Metric{&s}}

	var buf bytes.Buffer
	enc := NewSpackEncoder(context.Background(), CompressionNone, metrics)
	_, err := enc.Encode(&buf)
	require.NoError(t, err)

	body := buf.Bytes()
	body = body[HeaderSize:]

	// skip name pool "sensor\0" + value pool "mycounter\0"
	body = body[len("sensor\x00")+len("mycounter\x00"):]
	// skip common time (4) + common labels count (1)
	body = body[5:]

	expectMetric := []byte{
		0x0b, // types: typeCounter(2) << 2 | valueTypeManyWithTS(3) = 0b00001011
		0x00, // flags
		0x01, // labels count
		0x00, // sensor name idx
		0x00, // mycounter value idx
		0x01, // point count = 1
		// point: ts=100, value=42 (uint64 LE)
		0x64, 0x00, 0x00, 0x00,
		0x2a, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	}

	require.Equal(t, expectMetric, body)
}

// TestSeriesEncode_MixedWithScalar verifies that series and a regular scalar metric
// coexist in a single Metrics payload, with PointsCount header reflecting the sum.
func TestSeriesEncode_MixedWithScalar(t *testing.T) {
	g := NewGauge("scalar", 7)
	s := NewGaugeSeries("seriesg", []GaugePoint{
		{Timestamp: time.Unix(1, 0), Value: 1.0},
		{Timestamp: time.Unix(2, 0), Value: 2.0},
		{Timestamp: time.Unix(3, 0), Value: 3.0},
	})

	metrics := &Metrics{metrics: []Metric{&g, &s}}

	var buf bytes.Buffer
	enc := NewSpackEncoder(context.Background(), CompressionNone, metrics)
	_, err := enc.Encode(&buf)
	require.NoError(t, err)

	body := buf.Bytes()
	metricCount := binary.LittleEndian.Uint32(body[16:20])
	pointCount := binary.LittleEndian.Uint32(body[20:24])
	assert.Equal(t, uint32(2), metricCount, "metric count = 2 (scalar + series)")
	assert.Equal(t, uint32(4), pointCount, "point count = 1 + 3 = 4")
}

// TestSeriesEncode_V12 ensures version 1.2 layout works for series.
func TestSeriesEncode_V12(t *testing.T) {
	s := NewGaugeSeries("mygauge", []GaugePoint{
		{Timestamp: time.Unix(1000, 0), Value: 1.5},
	})

	metrics := &Metrics{metrics: []Metric{&s}}

	var buf bytes.Buffer
	enc := NewSpackEncoder(context.Background(), CompressionNone, metrics, WithVersion12())
	_, err := enc.Encode(&buf)
	require.NoError(t, err)

	body := buf.Bytes()
	// v1.2 header has version bytes 0x02 0x01
	assert.Equal(t, byte(0x02), body[2])
	assert.Equal(t, byte(0x01), body[3])
}

func TestCounterSeries_MarshalJSON(t *testing.T) {
	ts := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	c := NewCounterSeries("mycounter", []CounterPoint{
		{Timestamp: ts, Value: 42},
		{Timestamp: ts.Add(3 * time.Second), Value: 43},
		{Timestamp: ts.Add(5 * time.Second), Value: 47},
	}, WithUseNameTag(), WithTags(map[string]string{"ololo": "trololo"}))

	b, err := json.Marshal(&c)
	assert.NoError(t, err)

	expected := `{"type":"COUNTER","labels":{"name":"mycounter","ololo":"trololo"},"timeseries":[{"ts":1577836800,"value":42},{"ts":1577836803,"value":43},{"ts":1577836805,"value":47}]}`
	assert.Equal(t, expected, string(b))
}

func TestGaugeSeries_MarshalJSON(t *testing.T) {
	ts := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	c := NewGaugeSeries("mygauge", []GaugePoint{
		{Timestamp: ts, Value: 42.18},
		{Timestamp: ts.Add(3 * time.Second), Value: 43.18},
		{Timestamp: ts.Add(5 * time.Second), Value: 47.18},
	})

	b, err := json.Marshal(&c)
	assert.NoError(t, err)

	expected := `{"type":"DGAUGE","labels":{"sensor":"mygauge"},"timeseries":[{"ts":1577836800,"value":42.18},{"ts":1577836803,"value":43.18},{"ts":1577836805,"value":47.18}]}`
	assert.Equal(t, expected, string(b))
}
