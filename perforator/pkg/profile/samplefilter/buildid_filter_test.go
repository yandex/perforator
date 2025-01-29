package samplefilter

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yandex/perforator/perforator/pkg/profilequerylang"
)

func TestBuildBuildIDFilter(t *testing.T) {
	for _, test := range []struct {
		name   string
		query  string
		error  bool
		filter string
	}{
		{
			name:   "EmptySelector",
			query:  "{}",
			filter: "",
		},
		{
			name:   "UnrelatedMatchers",
			query:  "{env.foo=\"123\"}",
			filter: "",
		},
		{
			name:  "UnsupportedMultiValue",
			query: "{buildid=\"123|456\"}",
			error: true,
		},
		{
			name:  "UnsupportedNotEqual",
			query: "{buildid!=\"123\"}",
			error: true,
		},
		{
			name:   "OK",
			query:  "{buildid=\"123\"}",
			filter: "123",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			sel, err := profilequerylang.ParseSelector(test.query)
			require.NoError(t, err)
			f, err := BuildBuildIDFilter(sel)
			if test.error {
				assert.Error(t, err)
				return
			} else {
				if assert.NoError(t, err) {
					assert.Equal(t, buildIDFilter(test.filter), f)
				}
			}
		})
	}
}

func TestBuildIDFilterMatch(t *testing.T) {
	tests := []struct {
		name     string
		filter   string
		labels   map[string][]string
		expected bool
	}{
		{
			name:   "EmptyFilterMatchesAnyBuildID",
			filter: "",
			labels: map[string][]string{
				"buildid": []string{"123", "456"},
			},
			expected: true,
		},
		{
			name:   "EmptyFilterMatchesMissingBuildID",
			filter: "",
			labels: map[string][]string{
				"otherlabel": []string{"123"},
			},
			expected: true,
		},
		{
			name:   "SimpleMatch",
			filter: "123",
			labels: map[string][]string{
				"buildid": []string{"123"},
			},
			expected: true,
		},
		{
			name:   "SimpleNoMatch",
			filter: "123",
			labels: map[string][]string{
				"buildid": []string{"456"},
			},
			expected: false,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			f := buildIDFilter(test.filter)
			assert.Equal(t, test.expected, f.Matches(test.labels))
		})
	}
}
