package solomon

import (
	"testing"

	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"

	"github.com/yandex/perforator/library/go/ptr"
)

func TestPrometheusMetrics(t *testing.T) {
	gauge := NewGauge("subregister1_mygauge", 42, WithTags(map[string]string{"ololo": "trololo"}))
	counter := NewCounter("subregisters_count", 2, WithTags(map[string]string{}))
	hist := NewHistogram(
		"subregister1_subregister2_myhistogram",
		[]float64{1, 2, 3},
		[]int64{1, 2, 1},
		2,
		WithTags(map[string]string{"ololo": "trololo", "shimba": "boomba"}),
	)
	group1 := NewCounter("metrics_group", 2, WithTags(map[string]string{}))
	group2 := NewCounter("metrics_group", 3, WithTags(map[string]string{}))

	testCases := []struct {
		name      string
		metrics   []*dto.MetricFamily
		expect    *Metrics
		expectErr error
	}{
		{
			name: "success",
			metrics: []*dto.MetricFamily{
				{
					Name: ptr.String("subregister1_mygauge"),
					Help: ptr.String(""),
					Type: ptr.T(dto.MetricType_GAUGE),
					Metric: []*dto.Metric{
						{
							Label: []*dto.LabelPair{
								{Name: ptr.String("ololo"), Value: ptr.String("trololo")},
							},
							Gauge: &dto.Gauge{Value: ptr.Float64(42)},
						},
					},
				},
				{
					Name: ptr.String("subregisters_count"),
					Help: ptr.String(""),
					Type: ptr.T(dto.MetricType_COUNTER),
					Metric: []*dto.Metric{
						{
							Label:   []*dto.LabelPair{},
							Counter: &dto.Counter{Value: ptr.Float64(2)},
						},
					},
				},
				{
					Name: ptr.String("subregister1_subregister2_myhistogram"),
					Help: ptr.String(""),
					Type: ptr.T(dto.MetricType_HISTOGRAM),
					Metric: []*dto.Metric{
						{
							Label: []*dto.LabelPair{
								{Name: ptr.String("ololo"), Value: ptr.String("trololo")},
								{Name: ptr.String("shimba"), Value: ptr.String("boomba")},
							},
							Histogram: &dto.Histogram{
								SampleCount: ptr.Uint64(6),
								SampleSum:   ptr.Float64(4.2),
								Bucket: []*dto.Bucket{
									{CumulativeCount: ptr.Uint64(1), UpperBound: ptr.Float64(1)}, // 0.5 written
									{CumulativeCount: ptr.Uint64(3), UpperBound: ptr.Float64(2)}, // 1.5 & 1.7 written
									{CumulativeCount: ptr.Uint64(4), UpperBound: ptr.Float64(3)}, // 2.2 written
								},
							},
						},
					},
				},
				{
					Name: ptr.String("metrics_group"),
					Help: ptr.String(""),
					Type: ptr.T(dto.MetricType_COUNTER),
					Metric: []*dto.Metric{
						{
							Label:   []*dto.LabelPair{},
							Counter: &dto.Counter{Value: ptr.Float64(2)},
						},
						{
							Label:   []*dto.LabelPair{},
							Counter: &dto.Counter{Value: ptr.Float64(3)},
						},
					},
				},
			},
			expect: &Metrics{
				metrics: []Metric{
					&gauge,
					&counter,
					&hist,
					&group1,
					&group2,
				},
			},
			expectErr: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			s, err := PrometheusMetrics(tc.metrics)

			if tc.expectErr == nil {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tc.expectErr.Error())
			}

			assert.Equal(t, tc.expect, s)
		})
	}
}

func TestPrometheusSummaryMetric(t *testing.T) {
	src := []*dto.MetricFamily{
		{
			Name: ptr.String("subregister1_subregister2_mysummary"),
			Help: ptr.String(""),
			Type: func(mt dto.MetricType) *dto.MetricType { return &mt }(dto.MetricType_SUMMARY),
			Metric: []*dto.Metric{
				{
					Label: []*dto.LabelPair{
						{Name: ptr.String("ololo"), Value: ptr.String("trololo")},
						{Name: ptr.String("shimba"), Value: ptr.String("boomba")},
					},
					Summary: &dto.Summary{
						SampleCount: ptr.Uint64(8),
						SampleSum:   ptr.Float64(4.2),
						Quantile: []*dto.Quantile{
							{Value: ptr.Float64(1), Quantile: ptr.Float64(1)}, // 0.5 written
							{Value: ptr.Float64(3), Quantile: ptr.Float64(2)}, // 1.5 & 1.7 written
							{Value: ptr.Float64(4), Quantile: ptr.Float64(3)}, // 2.2 written
						},
					},
				},
			},
		},
	}

	mName := "subregister1_subregister2_mysummary"
	mTags := map[string]string{"ololo": "trololo", "shimba": "boomba"}

	hist := NewHistogram(mName, []float64{1, 2, 3}, []int64{1, 2, 1}, 4, WithTags(mTags))
	counter := NewCounter(mName+"_count", 8, WithTags(mTags))
	gauge := NewGauge(mName+"_sum", 4.2, WithTags(mTags))

	expect := &Metrics{
		metrics: []Metric{
			&hist,
			&counter,
			&gauge,
		},
	}

	s, err := PrometheusMetrics(src)
	assert.NoError(t, err)

	assert.Equal(t, expect, s)
}
