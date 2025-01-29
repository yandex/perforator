package samplefilter

import (
	"testing"

	pprof "github.com/google/pprof/profile"
	"github.com/stretchr/testify/require"

	"github.com/yandex/perforator/perforator/pkg/profilequerylang"
)

func TestBuildAndFilterByEnv(t *testing.T) {
	profiles := []*pprof.Profile{
		{
			Sample: []*pprof.Sample{
				{
					Label: map[string][]string{
						"env:key1": {"value1"},
					},
				},
				{
					Label: map[string][]string{
						"env:key2": {"value2"},
					},
				},
				{
					Label: map[string][]string{
						"env:key1": {"value1"},
						"env:key2": {"value2"},
					},
				},
			},
		},
		{
			Sample: []*pprof.Sample{
				{
					Label: map[string][]string{
						"env:key1": {"value1"},
					},
				},
				{
					Label: map[string][]string{
						"env:key1": {"value11"},
					},
				},
				{
					Label: map[string][]string{
						"env:key1": {"value11"},
						"env:key2": {"value22"},
					},
				},
			},
		},
		{
			Sample: []*pprof.Sample{
				{
					Label: map[string][]string{
						"env:key1": {"value1"},
						"key":      {"value"},
					},
				},
				{
					Label: nil,
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
			query: `{env.key1="value1"}`,
			expected: []*pprof.Profile{
				{
					Sample: []*pprof.Sample{
						{
							Label: map[string][]string{
								"env:key1": {"value1"},
							},
						},
						{
							Label: map[string][]string{
								"env:key1": {"value1"},
								"env:key2": {"value2"},
							},
						},
					},
				},
				{
					Sample: []*pprof.Sample{
						{
							Label: map[string][]string{
								"env:key1": {"value1"},
							},
						},
					},
				},
				{
					Sample: []*pprof.Sample{
						{
							Label: map[string][]string{
								"env:key1": {"value1"},
								"key":      {"value"},
							},
						},
					},
				},
			},
		},
		{
			name:  "non_env_keys",
			query: `{service="perforator|web-search", build_ids="a|b", tls.env1="value1"}`,
			expected: []*pprof.Profile{
				{
					Sample: []*pprof.Sample{
						{
							Label: map[string][]string{
								"env:key1": {"value1"},
							},
						},
						{
							Label: map[string][]string{
								"env:key1": {"value1"},
								"env:key2": {"value2"},
							},
						},
					},
				},
				{
					Sample: []*pprof.Sample{
						{
							Label: map[string][]string{
								"env:key1": {"value1"},
							},
						},
					},
				},
				{
					Sample: []*pprof.Sample{
						{
							Label: map[string][]string{
								"env:key1": {"value1"},
								"key":      {"value"},
							},
						},
					},
				},
			},
		},
		{
			name:  "several_env_match",
			query: `{service="perforator|web-search", build_ids="a|b", env.key1="value1", env.key2="value2"}`,
			expected: []*pprof.Profile{
				{
					Sample: []*pprof.Sample{
						{
							Label: map[string][]string{
								"env:key1": {"value1"},
								"env:key2": {"value2"},
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
			name:  "several_env_no_match",
			query: `{service="perforator|web-search", build_ids="a|b", env.key="value", env.key2="value2"}`,
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
			name:     "no_env_in_query",
			query:    `{service="perforator|web-search", build_ids="a|b"}`,
			expected: profiles,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			parsedSelector, _ := profilequerylang.ParseSelector(test.query)
			envFilter, err := BuildEnvFilter(parsedSelector)
			require.NoError(t, err)
			filteredProfiles := FilterProfilesBySampleFilters(profiles, envFilter)
			require.Equal(t, test.expected, filteredProfiles)
		})
	}
}
