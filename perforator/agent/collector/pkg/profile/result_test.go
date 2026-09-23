package profile

import (
	"slices"
	"testing"
	"time"

	pprof "github.com/google/pprof/profile"
	"github.com/stretchr/testify/require"

	"github.com/yandex/perforator/perforator/agent/collector/pkg/profileformat"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/profileresult"
	"github.com/yandex/perforator/perforator/pkg/profile/bundle"
)

func TestGetProfileEnvs(t *testing.T) {
	pprofProfile := &pprof.Profile{
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
					"env:key1": {"value3"},
					"env:key2": {"value4"},
				},
			},
		},
	}
	profile := &Profile{Profile: pprofProfile}
	envs := getProfileEnvs(profile)
	slices.Sort(envs)
	expected := []string{"key1=value1", "key2=value2", "key1=value3", "key2=value4"}
	slices.Sort(expected)
	require.Equal(t, expected, envs)
}

func TestToResult(t *testing.T) {
	for _, format := range []profileformat.ProfileFormat{profileformat.Pprof, profileformat.Yaprof} {
		t.Run(string(format), func(t *testing.T) {
			start := time.Unix(1700000000, 990000000)
			b := NewBuilder().AddSampleType("signal", "count")
			for i, pid := range []int64{42, 42, 43} {
				s := b.AddTimestampedSample(uint32(pid), start.Add(time.Duration(i)*time.Second)).
					AddValue(1).AddIntLabel("pid", pid, "").AddIntLabel("innermost_pidns_pid", 8, "id").AddIntLabel("size", 16, "bytes").
					AddStringLabel("env:key", "value").AddStringLabel("signal:name", "SIGINT").
					AddStringLabel("signal:name", "").AddStringLabel("custom", "label")
				s.AddNativeLocation(0x1100).SetMapping().SetBegin(0x1000).SetEnd(0x2000).
					SetBuildID("build-id").SetPath("/bin/test").Finish().Finish().Finish()
			}
			p := b.FinishRaw()
			p.PeriodType = &pprof.ValueType{}
			p.Comments = []string{"service:test"}
			r, err := p.ToResult(format)
			require.NoError(t, err)
			require.Equal(t, profileresult.Meta{
				BuildIDs: []string{"build-id"}, Envs: []string{"key=value"},
				EventTypes: []string{"signal.count"}, SignalTypes: []string{"SIGINT"},
				PIDSampleCounts: map[int64]int{42: 2, 43: 1}, SampleCount: 3,
				StartTimestamp: start, EndTimestamp: start.Add(2 * time.Second),
			}, r.Meta)
			decoded, err := r.ParsePprof()
			require.NoError(t, err)
			require.Len(t, decoded.Sample, 3)
			require.Equal(t, []string{"bytes"}, decoded.Sample[0].NumUnit["size"])
			if format == profileformat.Yaprof {
				// A native producer can supply just yaprof. Conversion aggregates
				// identical samples, but must preserve values and comments.
				native := &profileresult.Result{Bundle: bundle.NewYaprofBundle(r.Bundle.GetYaprof())}
				converted, err := native.ParsePprof()
				require.NoError(t, err)
				require.Len(t, converted.Sample, 2)
				require.Equal(t, start.UnixNano(), converted.TimeNanos)
				require.Equal(t, int64(2*time.Second), converted.DurationNanos)
				totals := map[int64]int64{}
				for _, sample := range converted.Sample {
					totals[sample.NumLabel["pid"][0]] += sample.Value[0]
				}
				require.Equal(t, map[int64]int64{42: 2, 43: 1}, totals)
				require.Contains(t, converted.Comments, "service:test")
				require.Equal(t, []string{"id"}, converted.Sample[0].NumUnit["innermost_pidns_pid"])
			}
			require.Contains(t, decoded.Comments, "service:test")
			require.Equal(t, start.UnixNano(), decoded.TimeNanos)
			require.Equal(t, int64(2*time.Second), decoded.DurationNanos)
			require.Equal(t, []string{"label"}, decoded.Sample[0].Label["custom"])
			require.Equal(t, []int64{8}, decoded.Sample[0].NumLabel["innermost_pidns_pid"])
			require.Equal(t, []string{"id"}, decoded.Sample[0].NumUnit["innermost_pidns_pid"])
			p.Sample[0].Value[0] = 999
			p.Mapping[0].BuildID = "changed"
			again, err := r.ParsePprof()
			require.NoError(t, err)
			require.Equal(t, decoded.Sample[0].Value, again.Sample[0].Value)
			require.Equal(t, []string{"build-id"}, r.Meta.BuildIDs)
		})
	}
}

func TestToResultErrors(t *testing.T) {
	for _, p := range []*Profile{nil, {}, {Profile: &pprof.Profile{Sample: []*pprof.Sample{{Value: []int64{1}}}}}} {
		_, err := p.ToResult(profileformat.Pprof)
		require.Error(t, err)
	}
	_, err := NewProfile().ToResult("unknown")
	require.Error(t, err)
}

func TestToResultEmptyAndPIDCounts(t *testing.T) {
	p := NewProfile()
	r, err := p.ToResult(profileformat.Pprof)
	require.NoError(t, err)
	require.Zero(t, r.Meta.SampleCount)
	require.True(t, r.Meta.StartTimestamp.IsZero())
	require.True(t, r.Meta.EndTimestamp.IsZero())
	require.Equal(t, []string{"cpu.cycles"}, r.Meta.EventTypes)
	p.SampleType = []*pprof.ValueType{{Type: "cpu", Unit: "cycles"}}
	for _, pids := range [][]int64{nil, {42}, {42, 43}} {
		p.Sample = append(p.Sample, &pprof.Sample{Value: []int64{1}, NumLabel: map[string][]int64{"pid": pids}})
	}
	r, err = p.ToResult(profileformat.Pprof)
	require.NoError(t, err)
	require.Equal(t, 3, r.Meta.SampleCount)
	require.Equal(t, map[int64]int{42: 1}, r.Meta.PIDSampleCounts)
}
