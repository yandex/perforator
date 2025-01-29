package samplefilter

import (
	"reflect"
	"testing"

	pprof "github.com/google/pprof/profile"
	"github.com/stretchr/testify/require"

	"github.com/yandex/perforator/perforator/pkg/profilequerylang"
)

func TestTlsFilterMatch(t *testing.T) {
	for _, test := range []struct {
		name       string
		query      string
		labels     map[string][]string
		buildError bool
		expected   bool
	}{
		{
			name:  "simple",
			query: `{tls.key="value"}`,
			labels: map[string][]string{
				"tls:perforator_tls_key": []string{"value"},
			},
			expected: true,
		},
		{
			name:  "non_tls_keys",
			query: `{service="perforator|web-search", build_ids="a|b", tls.key="value"}`,
			labels: map[string][]string{
				"tls:perforator_tls_key": []string{"value"},
			},
			expected: true,
		},
		{
			name:  "several_tls_match",
			query: `{service="perforator|web-search", build_ids="a|b", tls.key="value", tls.key2="value2"}`,
			labels: map[string][]string{
				"tls:perforator_tls_key":  []string{"value"},
				"tls:perforator_tls_key2": []string{"value2"},
			},
			expected: true,
		},
		{
			name:  "several_tls_no_match",
			query: `{service="perforator|web-search", build_ids="a|b", tls.key="value", tls.key2="value2"}`,
			labels: map[string][]string{
				"tls:perforator_tls_key":  []string{"value"},
				"tls:perforator_tls_key3": []string{"value3"},
			},
			expected: false,
		},
		{
			name:     "no_tls_label",
			query:    `{service="perforator|web-search", build_ids="a|b", tls.key="value"}`,
			labels:   map[string][]string{},
			expected: false,
		},
		{
			name:  "tls_inequality",
			query: `{service="perforator|web-search", build_ids="a|b", tls.key!="value"}`,
			labels: map[string][]string{
				"tls:perforator_tls_key": []string{"value"},
			},
			buildError: true,
		},
		{
			name:  "tls_or",
			query: `{service="perforator|web-search", build_ids="a|b", tls.key="value|value2"}`,
			labels: map[string][]string{
				"tls:perforator_tls_key": []string{"value"},
			},
			buildError: true,
		},
		{
			name:  "tls_numeric_key",
			query: `{service="perforator|web-search", build_ids="a|b", tls.2="X"}`,
			labels: map[string][]string{
				"tls:perforator_tls_2": []string{"X"},
			},
			expected: true,
		},
		{
			name:  "non_tls_labels",
			query: `{service="perforator|web-search", build_ids="a|b", tls.key="X"}`,
			labels: map[string][]string{
				"tls:perforator_tls_key": []string{"X"},
				"simple_label":           []string{"123"},
			},
			expected: true,
		},
		{
			name:  "legacy query with perforator_tls_ prefix",
			query: `{tls.perforator_tls_key="value"}`,
			labels: map[string][]string{
				"tls:perforator_tls_key": []string{"value"},
			},
			expected: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			parsedSelector, _ := profilequerylang.ParseSelector(test.query)
			tlsFilter, err := BuildTLSFilter(parsedSelector)

			if test.buildError {
				require.Error(t, err)
				return
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, test.expected, tlsFilter.Matches(test.labels))
		})
	}
}

func TestFilterSamplesByTLS(t *testing.T) {
	profiles := []*pprof.Profile{
		{
			Sample: []*pprof.Sample{
				{
					Label: map[string][]string{
						"tls:perforator_tls_key1": {"value1"},
					},
				},
				{
					Label: map[string][]string{
						"tls:perforator_tls_key2": {"value2"},
					},
				},
				{
					Label: map[string][]string{
						"tls:perforator_tls_key1": {"value1"},
						"tls:perforator_tls_key2": {"value2"},
					},
				},
			},
		},
		{
			Sample: []*pprof.Sample{
				{
					Label: map[string][]string{
						"tls:perforator_tls_key1": {"value1"},
					},
				},
				{
					Label: map[string][]string{
						"tls:perforator_tls_key1": {"value11"},
					},
				},
				{
					Label: map[string][]string{
						"tls:perforator_tls_key1": {"value11"},
						"tls:perforator_tls_key2": {"value22"},
					},
				},
			},
		},
	}

	filteredProfiles := FilterProfilesBySampleFilters(profiles, tlsFilter(map[string]string{
		"key1": "value1",
	}))
	require.Equal(t, []*pprof.Profile{
		{
			Sample: []*pprof.Sample{
				{
					Label: map[string][]string{
						"tls:perforator_tls_key1": {"value1"},
					},
				},
				{
					Label: map[string][]string{
						"tls:perforator_tls_key1": {"value1"},
						"tls:perforator_tls_key2": {"value2"},
					},
				},
			},
		},
		{
			Sample: []*pprof.Sample{
				{
					Label: map[string][]string{
						"tls:perforator_tls_key1": {"value1"},
					},
				},
			},
		},
	}, filteredProfiles)
	filteredProfiles = FilterProfilesBySampleFilters(profiles, tlsFilter(map[string]string{
		"key1": "value1",
		"key2": "value2",
	}))
	require.Equal(t, []*pprof.Profile{
		{
			Sample: []*pprof.Sample{
				{
					Label: map[string][]string{
						"tls:perforator_tls_key1": {"value1"},
						"tls:perforator_tls_key2": {"value2"},
					},
				},
			},
		},
		{
			Sample: []*pprof.Sample{},
		},
	}, filteredProfiles)
}

func samplesEqual(a, b *pprof.Sample) bool {
	return reflect.DeepEqual(a.Value, b.Value) && reflect.DeepEqual(a.Label, b.Label)
}

func profilesEqual(a, b *pprof.Profile) bool {
	if len(a.Sample) != len(b.Sample) {
		return false
	}
	for i := range a.Sample {
		if !samplesEqual(a.Sample[i], b.Sample[i]) {
			return false
		}
	}
	return true
}

func profileArraysEqual(a, b []*pprof.Profile) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !profilesEqual(a[i], b[i]) {
			return false
		}
	}
	return true
}

func TestBuildAndFilterByTLS(t *testing.T) {
	profiles := []*pprof.Profile{
		{
			Sample: []*pprof.Sample{
				{
					Label: map[string][]string{
						"tls:perforator_tls_key1": {"value1"},
					},
				},
				{
					Label: map[string][]string{
						"tls:perforator_tls_key2": {"value2"},
					},
				},
				{
					Label: map[string][]string{
						"tls:perforator_tls_key1": {"value1"},
						"tls:perforator_tls_key2": {"value2"},
					},
				},
			},
		},
		{
			Sample: []*pprof.Sample{
				{
					Label: map[string][]string{
						"tls:perforator_tls_key1": {"value1"},
					},
				},
				{
					Label: map[string][]string{
						"tls:perforator_tls_key1": {"value11"},
					},
				},
				{
					Label: map[string][]string{
						"tls:perforator_tls_key1": {"value11"},
						"tls:perforator_tls_key2": {"value22"},
					},
				},
			},
		},
		{
			Sample: []*pprof.Sample{
				{
					Label: map[string][]string{
						"tls:perforator_tls_key1": {"value1"},
						"key":                     {"value"},
					},
				},
			},
		},
	}
	for _, test := range []struct {
		name     string
		query    string
		expected []*pprof.Profile
	}{
		{
			name:  "simple",
			query: `{tls.key1="value1"}`,
			expected: []*pprof.Profile{
				{
					Sample: []*pprof.Sample{
						{
							Label: map[string][]string{
								"tls:perforator_tls_key1": {"value1"},
							},
						},
						{
							Label: map[string][]string{
								"tls:perforator_tls_key1": {"value1"},
								"tls:perforator_tls_key2": {"value2"},
							},
						},
					},
				},
				{
					Sample: []*pprof.Sample{
						{
							Label: map[string][]string{
								"tls:perforator_tls_key1": {"value1"},
							},
						},
					},
				},
				{
					Sample: []*pprof.Sample{
						{
							Label: map[string][]string{
								"tls:perforator_tls_key1": {"value1"},
								"key":                     {"value"},
							},
						},
					},
				},
			},
		},
		{
			name:  "non_tls_keys",
			query: `{service="perforator|web-search", build_ids="a|b", tls.key1="value1"}`,
			expected: []*pprof.Profile{
				{
					Sample: []*pprof.Sample{
						{
							Label: map[string][]string{
								"tls:perforator_tls_key1": {"value1"},
							},
						},
						{
							Label: map[string][]string{
								"tls:perforator_tls_key1": {"value1"},
								"tls:perforator_tls_key2": {"value2"},
							},
						},
					},
				},
				{
					Sample: []*pprof.Sample{
						{
							Label: map[string][]string{
								"tls:perforator_tls_key1": {"value1"},
							},
						},
					},
				},
				{
					Sample: []*pprof.Sample{
						{
							Label: map[string][]string{
								"tls:perforator_tls_key1": {"value1"},
								"key":                     {"value"},
							},
						},
					},
				},
			},
		},
		{
			name:  "several_tls_match",
			query: `{service="perforator|web-search", build_ids="a|b", tls.key1="value1", tls.key2="value2"}`,
			expected: []*pprof.Profile{
				{
					Sample: []*pprof.Sample{
						{
							Label: map[string][]string{
								"tls:perforator_tls_key1": {"value1"},
								"tls:perforator_tls_key2": {"value2"},
							},
						},
					},
				},
				{
					Sample: []*pprof.Sample{},
				},
				{
					Sample: []*pprof.Sample{},
				},
			},
		},
		{
			name:  "several_tls_no_match",
			query: `{service="perforator|web-search", build_ids="a|b", tls.key="value", tls.key2="value2"}`,
			expected: []*pprof.Profile{
				{
					Sample: []*pprof.Sample{},
				},
				{
					Sample: []*pprof.Sample{},
				},
				{
					Sample: []*pprof.Sample{},
				},
			},
		},
		{
			name:     "no_tls_in_query",
			query:    `{service="perforator|web-search", build_ids="a|b"}`,
			expected: profiles,
		},
		{
			name:  "legacy query with perforator_tls_ prefix",
			query: `{tls.perforator_tls_key1="value1"}`,
			expected: []*pprof.Profile{
				{
					Sample: []*pprof.Sample{
						{
							Label: map[string][]string{
								"tls:perforator_tls_key1": {"value1"},
							},
						},
						{
							Label: map[string][]string{
								"tls:perforator_tls_key1": {"value1"},
								"tls:perforator_tls_key2": {"value2"},
							},
						},
					},
				},
				{
					Sample: []*pprof.Sample{
						{
							Label: map[string][]string{
								"tls:perforator_tls_key1": {"value1"},
							},
						},
					},
				},
				{
					Sample: []*pprof.Sample{
						{
							Label: map[string][]string{
								"tls:perforator_tls_key1": {"value1"},
								"key":                     {"value"},
							},
						},
					},
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			parsedSelector, _ := profilequerylang.ParseSelector(test.query)
			tlsFilter, err := BuildTLSFilter(parsedSelector)
			require.NoError(t, err)
			profilesCopy := make([]*pprof.Profile, 0, len(profiles))
			for _, p := range profiles {
				profilesCopy = append(profilesCopy, p.Copy())
			}
			require.True(t, profileArraysEqual(profiles, profilesCopy))

			filteredProfiles := FilterProfilesBySampleFilters(profilesCopy, tlsFilter)
			require.True(t, profileArraysEqual(test.expected, filteredProfiles))
		})
	}
}
