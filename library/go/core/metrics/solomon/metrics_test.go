package solomon

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/yandex/perforator/library/go/core/metrics"
)

func TestMetrics_MarshalJSON(t *testing.T) {
	counter := NewCounter("mycounter", 42, WithTags(map[string]string{"ololo": "trololo"}))
	ratedCounter := NewCounter("myratedcounter", 42, WithTags(map[string]string{"ololo": "trololo"}), WithRated(true))
	gauge := NewGauge("mygauge", 14.89, WithTags(map[string]string{"shimba": "boomba"}))
	gaugeWithTS := NewGauge("mygauge", 42.24, WithTags(map[string]string{"shimba": "boomba"}), WithTimestamp(time.Unix(1500000000, 0)))
	timer := NewTimer("mytimer", 1456*time.Millisecond, WithTags(map[string]string{"looken": "tooken"}))
	hist := NewHistogram("myhistogram", []float64{1, 2, 3}, []int64{1, 2, 1}, 1, WithTags(map[string]string{"chicken": "cooken"}))
	ratedHist := NewHistogram("myratedhistogram", []float64{1, 2, 3}, []int64{1, 2, 1}, 1, WithTags(map[string]string{"chicken": "cooken"}), WithRated(true))

	s := &Metrics{
		metrics: []Metric{&counter, &ratedCounter, &gauge, &gaugeWithTS, &timer, &hist, &ratedHist},
	}

	b, err := json.Marshal(s)
	assert.NoError(t, err)

	expected := []byte(`{"metrics":[` +
		`{"type":"COUNTER","labels":{"ololo":"trololo","sensor":"mycounter"},"value":42},` +
		`{"type":"RATE","labels":{"ololo":"trololo","sensor":"myratedcounter"},"value":42},` +
		`{"type":"DGAUGE","labels":{"sensor":"mygauge","shimba":"boomba"},"value":14.89},` +
		`{"type":"DGAUGE","labels":{"sensor":"mygauge","shimba":"boomba"},"value":42.24,"ts":1500000000},` +
		`{"type":"DGAUGE","labels":{"looken":"tooken","sensor":"mytimer"},"value":1.456},` +
		`{"type":"HIST","labels":{"chicken":"cooken","sensor":"myhistogram"},"hist":{"bounds":[1,2,3],"buckets":[1,2,1],"inf":1}},` +
		`{"type":"HIST_RATE","labels":{"chicken":"cooken","sensor":"myratedhistogram"},"hist":{"bounds":[1,2,3],"buckets":[1,2,1],"inf":1}}` +
		`]}`)
	assert.Equal(t, expected, b)
}

func timeAsRef(t time.Time) *time.Time {
	return &t
}

func TestMetrics_with_timestamp_MarshalJSON(t *testing.T) {
	counter := NewCounter("mycounter", 42, WithTags(map[string]string{"ololo": "trololo"}))
	gauge := NewGauge("mygauge", 42.24, WithTags(map[string]string{"oki": "toki"}), WithTimestamp(time.Unix(1500000000, 0)))

	s := &Metrics{
		metrics:   []Metric{&counter, &gauge},
		timestamp: timeAsRef(time.Unix(1657710477, 0)),
	}

	b, err := json.Marshal(s)
	assert.NoError(t, err)

	expected := []byte(`{"metrics":[` +
		`{"type":"COUNTER","labels":{"ololo":"trololo","sensor":"mycounter"},"value":42},` +
		`{"type":"DGAUGE","labels":{"oki":"toki","sensor":"mygauge"},"value":42.24,"ts":1500000000}` +
		`],"ts":1657710477}`)
	assert.Equal(t, expected, b)
}

func TestMetrics_with_common_labels_MarshalJSON(t *testing.T) {
	counter := NewCounter("mycounter", 42, WithTags(map[string]string{"ololo": "trololo"}))
	gauge := NewGauge("mygauge", 42.24, WithTags(map[string]string{"oki": "toki"}))
	s := &Metrics{
		metrics:      []Metric{&counter, &gauge},
		commonLabels: map[string]string{"host": "myhost"},
	}

	b, err := json.Marshal(s)
	assert.NoError(t, err)

	expected := []byte(`{"metrics":[` +
		`{"type":"COUNTER","labels":{"ololo":"trololo","sensor":"mycounter"},"value":42},` +
		`{"type":"DGAUGE","labels":{"oki":"toki","sensor":"mygauge"},"value":42.24}` +
		`],"commonLabels":{"host":"myhost"}}`)
	assert.Equal(t, expected, b)
}

func TestRated(t *testing.T) {
	counter := NewCounter("mycounter", 42, WithTags(map[string]string{"ololo": "trololo"}))
	counterRated := NewCounter("mycounter", 42, WithTags(map[string]string{"ololo": "trololo"}), WithRated(true))
	gauge := NewGauge("mygauge", 42, WithTags(map[string]string{"ololo": "trololo"}))
	gaugeUnchanged := NewGauge("mygauge", 42, WithTags(map[string]string{"ololo": "trololo"}))
	timer := NewTimer("mytimer", 1*time.Second, WithTags(map[string]string{"ololo": "trololo"}))
	timerUnchanged := NewTimer("mytimer", 1*time.Second, WithTags(map[string]string{"ololo": "trololo"}))
	hist := NewHistogram("myhistogram", []float64{1, 2, 3}, nil, 0, WithTags(map[string]string{"ololo": "trololo"}))
	histRated := NewHistogram("myhistogram", []float64{1, 2, 3}, nil, 0, WithTags(map[string]string{"ololo": "trololo"}), WithRated(true))
	ifaceCounter := NewCounter("mycounter", 42, WithTags(map[string]string{"ololo": "trololo"}))
	ifaceCounterRated := NewCounter("mycounter", 42, WithTags(map[string]string{"ololo": "trololo"}), WithRated(true))

	testCases := []struct {
		name     string
		s        any
		expected Metric
	}{
		{"counter", &counter, &counterRated},
		{"gauge", &gauge, &gaugeUnchanged},
		{"timer", &timer, &timerUnchanged},
		{"histogram", &hist, &histRated},
		{"metric_interface", metrics.Counter(&ifaceCounter), &ifaceCounterRated},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			Rated(tc.s)
			assert.Equal(t, tc.expected, tc.s)
		})
	}
}

func TestSplitToChunks(t *testing.T) {
	mk := func(name string) *Counter {
		c := NewCounter(name, 0)
		return &c
	}

	zeroMetrics := Metrics{
		metrics: []Metric{},
	}
	oneMetric := Metrics{
		metrics: []Metric{mk("a")},
	}
	twoMetrics := Metrics{
		metrics: []Metric{mk("a"), mk("b")},
	}
	fourMetrics := Metrics{
		metrics: []Metric{mk("a"), mk("b"), mk("c"), mk("d")},
	}
	fiveMetrics := Metrics{
		metrics: []Metric{mk("a"), mk("b"), mk("c"), mk("d"), mk("e")},
	}

	chunks := zeroMetrics.SplitToChunks(2)
	assert.Equal(t, 1, len(chunks))
	assert.Equal(t, 0, len(chunks[0].metrics))

	chunks = oneMetric.SplitToChunks(1)
	assert.Equal(t, 1, len(chunks))
	assert.Equal(t, 1, len(chunks[0].metrics))
	assert.Equal(t, "a", chunks[0].metrics[0].Name())

	chunks = oneMetric.SplitToChunks(2)
	assert.Equal(t, 1, len(chunks))
	assert.Equal(t, 1, len(chunks[0].metrics))
	assert.Equal(t, "a", chunks[0].metrics[0].Name())

	chunks = twoMetrics.SplitToChunks(1)
	assert.Equal(t, 2, len(chunks))
	assert.Equal(t, 1, len(chunks[0].metrics))
	assert.Equal(t, 1, len(chunks[1].metrics))
	assert.Equal(t, "a", chunks[0].metrics[0].Name())
	assert.Equal(t, "b", chunks[1].metrics[0].Name())

	chunks = twoMetrics.SplitToChunks(2)
	assert.Equal(t, 1, len(chunks))
	assert.Equal(t, 2, len(chunks[0].metrics))
	assert.Equal(t, "a", chunks[0].metrics[0].Name())
	assert.Equal(t, "b", chunks[0].metrics[1].Name())

	chunks = fourMetrics.SplitToChunks(2)
	assert.Equal(t, 2, len(chunks))
	assert.Equal(t, 2, len(chunks[0].metrics))
	assert.Equal(t, 2, len(chunks[1].metrics))
	assert.Equal(t, "a", chunks[0].metrics[0].Name())
	assert.Equal(t, "b", chunks[0].metrics[1].Name())
	assert.Equal(t, "c", chunks[1].metrics[0].Name())
	assert.Equal(t, "d", chunks[1].metrics[1].Name())

	chunks = fiveMetrics.SplitToChunks(2)
	assert.Equal(t, 3, len(chunks))
	assert.Equal(t, 2, len(chunks[0].metrics))
	assert.Equal(t, 2, len(chunks[1].metrics))
	assert.Equal(t, 1, len(chunks[2].metrics))
	assert.Equal(t, "a", chunks[0].metrics[0].Name())
	assert.Equal(t, "b", chunks[0].metrics[1].Name())
	assert.Equal(t, "c", chunks[1].metrics[0].Name())
	assert.Equal(t, "d", chunks[1].metrics[1].Name())
	assert.Equal(t, "e", chunks[2].metrics[0].Name())

	chunks = fiveMetrics.SplitToChunks(0)
	assert.Equal(t, 1, len(chunks))
	assert.Equal(t, 5, len(chunks[0].metrics))
	assert.Equal(t, "a", chunks[0].metrics[0].Name())
	assert.Equal(t, "b", chunks[0].metrics[1].Name())
	assert.Equal(t, "c", chunks[0].metrics[2].Name())
	assert.Equal(t, "d", chunks[0].metrics[3].Name())
	assert.Equal(t, "e", chunks[0].metrics[4].Name())
}
