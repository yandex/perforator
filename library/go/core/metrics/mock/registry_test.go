package mock

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGaugeWithInitialValue(t *testing.T) {
	for _, initialValue := range []float64{0, 17.5, -17.5} {
		t.Run(fmt.Sprint(initialValue), func(t *testing.T) {
			opts := NewRegistryOpts()
			opts.AllowLoadRegisteredMetrics = true
			r := NewRegistry(opts)
			scope := r.WithPrefix("worker").WithTags(map[string]string{"job_index": "1"}).(*Registry)
			gauge := scope.GaugeWithInitialValue("lag", initialValue).(*Gauge)
			require.Equal(t, initialValue, gauge.Value.Load())
			require.Equal(t, "worker.lag", gauge.Name)
			require.Equal(t, map[string]string{"job_index": "1"}, gauge.Tags)
			gauge.Add(1)
			existing := scope.GaugeWithInitialValue("lag", initialValue+100).(*Gauge)
			require.Same(t, gauge, existing)
			require.Equal(t, initialValue+1, existing.Value.Load())
			existing.Set(42)
			require.Equal(t, float64(42), gauge.Value.Load())
			require.Same(t, gauge, scope.Gauge("lag"))
			require.Equal(t, float64(42), gauge.Value.Load())
			require.Equal(t, float64(0), r.Gauge("default").(*Gauge).Value.Load())
		})
	}

	t.Run("DuplicateRegistrationPolicy", func(t *testing.T) {
		r := NewRegistry(nil)
		r.GaugeWithInitialValue("lag", 17)
		require.Panics(t, func() { r.GaugeWithInitialValue("lag", 23) })
	})
}

func TestGetWithTags(t *testing.T) {
	var expected = int64(3)

	opts := NewRegistryOpts()
	opts.AllowLoadRegisteredMetrics = true
	r := NewRegistry(opts)

	tags := map[string]string{
		"ololo": "trololo",
	}

	for i := 0; i < int(expected); i++ {
		r.WithTags(tags).Counter("counter").Inc()
	}

	rt, ok := r.GetWithTags(tags)
	require.True(t, ok)
	counter, ok := rt.GetCounter("counter")
	require.True(t, ok)
	require.Equal(t, expected, counter.Value.Load())

	_, ok = rt.GetCounter("counter1")
	require.False(t, ok)
}
