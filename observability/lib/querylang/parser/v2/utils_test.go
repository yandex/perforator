package parserv2_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	parserv2 "github.com/yandex/perforator/observability/lib/querylang/parser/v2"
)

func TestParseSolomonDuration(t *testing.T) {
	tests := []struct {
		duration string
		expected time.Duration
		err      bool
	}{
		{
			duration: "10s",
			expected: 10 * time.Second,
		},
		{
			duration: "5m",
			expected: 5 * time.Minute,
		},
		{
			duration: "5m30s",
			expected: 5*time.Minute + 30*time.Second,
		},
		{
			duration: "2h",
			expected: 2 * time.Hour,
		},
		{
			duration: "1d",
			expected: 24 * time.Hour,
		},
		{
			duration: "invalid",
			err:      true,
		},
		{
			duration: "1d40x",
			err:      true,
		},
		{
			duration: "1d40",
			err:      true,
		},
		{
			duration: "10x",
			err:      true,
		},
		{
			duration: "10",
			err:      true,
		},
	}

	for _, test := range tests {
		t.Run(test.duration, func(t *testing.T) {
			actual, err := parserv2.ParseSolomonDuration(test.duration)
			if test.err {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, test.expected, actual)
		})
	}
}
