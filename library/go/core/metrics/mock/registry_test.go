package mock

import (
	"testing"

	"github.com/stretchr/testify/require"
)

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
