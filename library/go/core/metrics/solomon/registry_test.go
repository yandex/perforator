package solomon

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/atomic"
)

func TestRegistry_Gather(t *testing.T) {
	counter := NewCounter("myprefix.mycounter", 42, WithTags(map[string]string{"ololo": "trololo"}))
	gauge := NewGauge("myprefix.mygauge", 14.89, WithTags(map[string]string{"shimba": "boomba"}))
	timer := NewTimer("myprefix.mytimer", 1456*time.Millisecond, WithTags(map[string]string{"looken": "tooken"}))
	hist := NewHistogram("myprefix.myhistogram", []float64{1, 2, 3}, []int64{1, 2, 1}, 1, WithTags(map[string]string{"chicken": "cooken"}))

	r := &Registry{
		separator:     ".",
		prefix:        "myprefix",
		tags:          make(map[string]string),
		subregistries: make(map[string]*Registry),
		metrics: func() *sync.Map {
			metrics := map[string]Metric{
				"myprefix.mycounter":   &counter,
				"myprefix.mygauge":     &gauge,
				"myprefix.mytimer":     &timer,
				"myprefix.myhistogram": &hist,
			}

			sm := new(sync.Map)
			for k, v := range metrics {
				sm.Store(k, v)
			}

			return sm
		}(),
	}

	s, err := r.Gather()
	assert.NoError(t, err)

	expected := &Metrics{commonStartTime: r.startTime}
	r.metrics.Range(func(_, s any) bool {
		expected.metrics = append(expected.metrics, s.(Metric))
		return true
	})

	opts := cmp.Options{
		cmp.AllowUnexported(Metrics{}, baseMetric{}, Counter{}, Gauge{}, Timer{}, Histogram{}),
		cmpopts.IgnoreUnexported(sync.Mutex{}, atomic.Duration{}, atomic.Int64{}, atomic.Float64{}),
		// this will sort both slices for latest tests as well
		cmpopts.SortSlices(func(x, y Metric) bool {
			return x.Name() < y.Name()
		}),
	}

	assert.True(t, cmp.Equal(expected, s, opts...), cmp.Diff(expected, s, opts...))

	for _, sen := range s.metrics {
		var expectedMetric Metric
		for _, expSen := range expected.metrics {
			if expSen.Name() == sen.Name() {
				expectedMetric = expSen
				break
			}
		}
		require.NotNil(t, expectedMetric)

		assert.NotEqual(t, fmt.Sprintf("%p", expectedMetric), fmt.Sprintf("%p", sen))
		assert.IsType(t, expectedMetric, sen)

		switch st := sen.(type) {
		case *Counter:
			assert.NotEqual(t, fmt.Sprintf("%p", expectedMetric.(*Counter)), fmt.Sprintf("%p", st))
		case *Gauge:
			assert.NotEqual(t, fmt.Sprintf("%p", expectedMetric.(*Gauge)), fmt.Sprintf("%p", st))
		case *Timer:
			assert.NotEqual(t, fmt.Sprintf("%p", expectedMetric.(*Timer)), fmt.Sprintf("%p", st))
		case *Histogram:
			assert.NotEqual(t, fmt.Sprintf("%p", expectedMetric.(*Histogram)), fmt.Sprintf("%p", st))
		default:
			t.Fatalf("unexpected metric type: %T", sen)
		}
	}
}

func TestRegistryStartTimes(t *testing.T) {
	root := NewRegistry(NewRegistryOpts().SetTrackRateStartTime(true))
	root.startTime = 42
	r := root.Rated(true).(*Registry)

	before := uint32(time.Now().Unix())
	r.Counter("counter")
	histogram := NewHistogram("histogram", nil, nil, 0, WithRated(true))
	r.AddMetric(&histogram)
	gauge := NewGauge("gauge", 0)
	r.AddMetric(&gauge)
	explicit := NewCounter("explicit", 0, WithRated(true), WithStartTime(7))
	r.AddMetric(&explicit)
	after := uint32(time.Now().Unix())

	got, err := r.Gather()
	require.NoError(t, err)
	require.Equal(t, uint32(42), got.commonStartTime)

	for _, metric := range got.List() {
		startTime := metric.getStartTime()
		switch metric.Name() {
		case "counter", "histogram":
			require.GreaterOrEqual(t, startTime, before)
			require.LessOrEqual(t, startTime, after)
		case "explicit":
			require.Equal(t, uint32(7), startTime)
		case "gauge":
			require.Zero(t, startTime)
		}
	}
}

func TestRegistryTrackRateStartTime(t *testing.T) {
	before := uint32(time.Now().Unix())
	r := NewRegistry(NewRegistryOpts().SetTrackRateStartTime(true))
	counter := r.Rated(false).Counter("counter").(Metric)
	funcCounter := r.FuncCounter("func_counter", func() int64 { return 0 }).(Metric)
	histogram := NewHistogram("histogram", nil, nil, 0)
	r.AddMetric(&histogram)
	gauge := r.Gauge("gauge").(Metric)
	after := uint32(time.Now().Unix())

	for _, metric := range []Metric{counter, funcCounter, &histogram} {
		startTime := metric.getStartTime()
		require.GreaterOrEqual(t, startTime, before)
		require.LessOrEqual(t, startTime, after)
	}
	require.Zero(t, gauge.getStartTime())

	startTime := counter.getStartTime()
	Rated(counter)
	require.Equal(t, typeRated, counter.getType())
	require.Equal(t, startTime, counter.getStartTime())

	plain := NewRegistry(NewRegistryOpts()).Counter("plain").(Metric)
	require.Zero(t, plain.getStartTime())
}

func TestDoubleRegistration(t *testing.T) {
	r := NewRegistry(NewRegistryOpts())

	c0 := r.Counter("counter")
	c1 := r.Counter("counter")
	require.Equal(t, c0, c1)

	g0 := r.Gauge("counter")
	g1 := r.Gauge("counter")
	require.Equal(t, g0, g1)

	c2 := r.Counter("counter")
	require.NotEqual(t, reflect.ValueOf(c0).Elem().UnsafeAddr(), reflect.ValueOf(c2).Elem().UnsafeAddr())
}

func TestSubregistry(t *testing.T) {
	r := NewRegistry(NewRegistryOpts())

	r0 := r.WithPrefix("one")
	r1 := r0.WithPrefix("two")
	r2 := r0.WithTags(map[string]string{"foo": "bar"})

	_ = r0.Counter("counter")
	_ = r1.Counter("counter")
	_ = r2.Counter("counter")
}

func TestSubregistry_TagAndPrefixReorder(t *testing.T) {
	r := NewRegistry(NewRegistryOpts())

	r0 := r.WithPrefix("one")
	r1 := r.WithTags(map[string]string{"foo": "bar"})

	r3 := r0.WithTags(map[string]string{"foo": "bar"})
	r4 := r1.WithPrefix("one")

	require.True(t, r3 == r4)
}

func TestRatedRegistry(t *testing.T) {
	r := NewRegistry(NewRegistryOpts().SetRated(true))
	s := r.Counter("counter")
	b, _ := json.Marshal(s)
	expected := []byte(`{"type":"RATE","labels":{"sensor":"counter"},"value":0}`)
	assert.Equal(t, expected, b)
}

func TestNameTagRegistry(t *testing.T) {
	r := NewRegistry(NewRegistryOpts().SetUseNameTag(true))
	s := r.Counter("counter")

	b, _ := json.Marshal(s)
	expected := []byte(`{"type":"COUNTER","labels":{"name":"counter"},"value":0}`)
	assert.Equal(t, expected, b)

	sr := r.WithTags(map[string]string{"foo": "bar"})
	ssr := sr.Counter("sub_counter")

	b1, _ := json.Marshal(ssr)
	expected1 := []byte(`{"type":"COUNTER","labels":{"foo":"bar","name":"sub_counter"},"value":0}`)
	assert.Equal(t, expected1, b1)
}

func TestMetricFlags(t *testing.T) {
	r := NewRegistry(NewRegistryOpts())
	s := MemOnly(Rated(r.Counter("counter")))
	assert.Equal(t, s.(Metric).isMemOnly(), true)
	assert.Equal(t, s.(Metric).getType(), typeRated)
}
