package client

import (
	"slices"
	"testing"

	"github.com/google/pprof/profile"
	"github.com/stretchr/testify/require"
)

func TestGetProfileEnvs(t *testing.T) {
	profile := &profile.Profile{
		Sample: []*profile.Sample{
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
					"env:key1": {"value3"},
					"env:key2": {"value4"},
				},
			},
		},
	}
	envs := getProfileEnvs(profile)
	slices.Sort(envs)
	expected := []string{"key1=value1", "key2=value2", "key1=value3", "key2=value4"}
	slices.Sort(expected)
	require.Equal(t, expected, envs)
}
