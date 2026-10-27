package solomon

import (
	"fmt"

	dto "github.com/prometheus/client_model/go"
)

// PrometheusMetrics converts Prometheus metrics to Solomon metrics.
func PrometheusMetrics(metrics []*dto.MetricFamily) (*Metrics, error) {
	s := &Metrics{
		metrics: make([]Metric, 0, len(metrics)),
	}

	if len(metrics) == 0 {
		return s, nil
	}

	for _, mf := range metrics {
		if len(mf.Metric) == 0 {
			continue
		}

		for _, metric := range mf.Metric {

			tags := make(map[string]string, len(metric.Label))
			for _, label := range metric.Label {
				tags[label.GetName()] = label.GetValue()
			}

			switch *mf.Type {
			case dto.MetricType_COUNTER:
				counter := NewCounter(mf.GetName(), int64(metric.Counter.GetValue()), WithTags(tags))
				s.metrics = append(s.metrics, &counter)
			case dto.MetricType_GAUGE:
				gauge := NewGauge(mf.GetName(), metric.Gauge.GetValue(), WithTags(tags))
				s.metrics = append(s.metrics, &gauge)
			case dto.MetricType_HISTOGRAM:
				bounds := make([]float64, 0, len(metric.Histogram.Bucket))
				values := make([]int64, 0, len(metric.Histogram.Bucket))

				var prevValuesSum int64

				for _, bucket := range metric.Histogram.Bucket {
					// prometheus uses cumulative buckets where solomon uses instant
					bucketValue := int64(bucket.GetCumulativeCount())
					bucketValue -= prevValuesSum
					prevValuesSum += bucketValue

					bounds = append(bounds, bucket.GetUpperBound())
					values = append(values, bucketValue)
				}

				histogram := NewHistogram(
					mf.GetName(),
					bounds,
					values,
					int64(metric.Histogram.GetSampleCount())-prevValuesSum,
					WithTags(tags),
				)
				s.metrics = append(s.metrics, &histogram)
			case dto.MetricType_SUMMARY:
				bounds := make([]float64, 0, len(metric.Summary.Quantile))
				values := make([]int64, 0, len(metric.Summary.Quantile))

				var prevValuesSum int64

				for _, bucket := range metric.Summary.GetQuantile() {
					// prometheus uses cumulative buckets where solomon uses instant
					bucketValue := int64(bucket.GetValue())
					bucketValue -= prevValuesSum
					prevValuesSum += bucketValue

					bounds = append(bounds, bucket.GetQuantile())
					values = append(values, bucketValue)
				}

				mName := mf.GetName()

				histogram := NewHistogram(
					mName,
					bounds,
					values,
					int64(*metric.Summary.SampleCount)-prevValuesSum,
					WithTags(tags),
				)
				counter := NewCounter(mName+"_count", int64(*metric.Summary.SampleCount), WithTags(tags))
				gauge := NewGauge(mName+"_sum", *metric.Summary.SampleSum, WithTags(tags))

				s.metrics = append(
					s.metrics,
					&histogram,
					&counter,
					&gauge,
				)
			default:
				return nil, fmt.Errorf("unsupported type: %s", mf.Type.String())
			}
		}
	}

	return s, nil
}
