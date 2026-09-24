package profiler

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/yandex/perforator/perforator/agent/collector/pkg/profile"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/profileformat"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/profileresult"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/storage/client"
	"github.com/yandex/perforator/perforator/pkg/linux"
	"github.com/yandex/perforator/perforator/pkg/profile/bundle"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

func TestFlushSeparatesSampleTypes(t *testing.T) {
	for _, format := range []profileformat.ProfileFormat{profileformat.Pprof, profileformat.Yaprof} {
		t.Run(string(format), func(t *testing.T) {
			b := newMultiProfileBuilder(map[string]string{"service": "test"}, format)
			type expectedProfile struct {
				sampleTypes []profile.SampleType
				values      []int64
			}
			cases := []expectedProfile{
				{[]profile.SampleType{{Kind: "cpu", Unit: "cycles"}}, []int64{7}},
				{[]profile.SampleType{{Kind: "cpu", Unit: "nanoseconds"}}, []int64{19}},
				{[]profile.SampleType{{Kind: "cpu", Unit: "cycles"}, {Kind: "wall", Unit: "seconds"}}, []int64{7, 19}},
				{[]profile.SampleType{{Kind: "wall", Unit: "seconds"}, {Kind: "cpu", Unit: "cycles"}}, []int64{19, 7}},
			}
			for iteration := range 2 {
				start := time.Unix(1700000000+int64(iteration)*10, 0)
				expected := make(map[string]expectedProfile)
				for _, test := range cases {
					sampleTypes := test.sampleTypes
					eventTypes := make([]string, 0, len(sampleTypes))
					for _, st := range sampleTypes {
						eventTypes = append(eventTypes, st.Kind+"."+st.Unit)
					}
					expected[strings.Join(eventTypes, ",")] = test
					builder := b.EnsureBuilder("same-name", sampleTypes)
					require.Same(t, builder, b.EnsureBuilder("same-name", sampleTypes))
					s := builder.AddTimestampedSample(linux.ProcessKey{Pid: 42}, start).AddIntLabel("pid", 42, "")
					for _, value := range test.values {
						s.AddValue(value)
					}
					s.AddNativeLocation(0x1000).Finish().Finish()
				}
				flushed := b.RestartProfiles()
				require.Len(t, flushed.Profiles, 4)
				require.Equal(t, map[string]string{"service": "test"}, flushed.Labels)
				for _, r := range flushed.Profiles {
					require.Equal(t, 1, r.Meta.SampleCount)
					require.Equal(t, map[int64]int{42: 1}, r.Meta.PIDSampleCounts)
					require.Equal(t, start, r.Meta.StartTimestamp)
					key := strings.Join(r.Meta.EventTypes, ",")
					require.Contains(t, expected, key)
					want := expected[key]
					delete(expected, key)
					checkValues := func(result *profileresult.Result) {
						t.Helper()
						p, err := result.ParsePprof()
						require.NoError(t, err)
						require.NoError(t, p.CheckValid())
						require.Contains(t, p.Comments, "service:test")
						actualTypes := make([]profile.SampleType, 0, len(p.SampleType))
						for _, st := range p.SampleType {
							actualTypes = append(actualTypes, profile.SampleType{Kind: st.Type, Unit: st.Unit})
						}
						require.Equal(t, want.sampleTypes, actualTypes)
						require.Len(t, p.Sample, 1)
						require.Equal(t, want.values, p.Sample[0].Value)
					}
					checkValues(r)
					if format == profileformat.Yaprof {
						// Decode the actual yaprof payload, without falling back to retained pprof.
						checkValues(&profileresult.Result{Bundle: bundle.NewYaprofBundle(r.Bundle.GetYaprof())})
					}
				}
				require.Empty(t, expected)
			}
		})
	}
}

type sampleStoredListener map[linux.CurrentNamespacePID]int

func (l sampleStoredListener) OnSampleStored(pid linux.CurrentNamespacePID) { l[pid]++ }

type failingProfileStorage struct{ *client.InMemoryStorage }

func (*failingProfileStorage) StoreProfile(context.Context, client.LabeledProfile) error {
	return errors.New("storage unavailable")
}

func TestSaveSerializedProfileCounts(t *testing.T) {
	for _, test := range []struct {
		format    profileformat.ProfileFormat
		count     int
		pidCounts map[int64]int
		stored    sampleStoredListener
	}{
		{profileformat.Pprof, 2, map[int64]int{42: 1, 43: 1}, sampleStoredListener{42: 1, 43: 1}},
		{profileformat.Yaprof, 3, map[int64]int{42: 2, 43: 1}, sampleStoredListener{42: 2, 43: 1}},
	} {
		t.Run(string(test.format), func(t *testing.T) {
			b := newMultiProfileBuilder(nil, test.format)
			builder := b.EnsureBuilder("cpu", []profile.SampleType{{Kind: "cpu", Unit: "cycles"}})
			start := time.Unix(1700000000, 0)
			for _, pid := range []int64{42, 42, 43} {
				builder.AddTimestampedSample(linux.ProcessKey{Pid: uint32(pid)}, start).AddValue(1).
					AddIntLabel("pid", pid, "").AddNativeLocation(0x1000).Finish().Finish()
			}
			flushed := b.RestartProfiles()
			require.Len(t, flushed.Profiles, 1)
			r := flushed.Profiles[0]
			// pprof compacts duplicates in Finish; yaprof counts them before conversion.
			require.Equal(t, test.count, r.Meta.SampleCount)
			require.Equal(t, test.pidCounts, r.Meta.PIDSampleCounts)

			storage := client.NewInMemoryStorage(&client.InMemoryStorageConfig{})
			listener := sampleStoredListener{}
			p := &Profiler{storage: storage, eventListener: listener, log: xlog.ForTest(t).Logger()}
			result := client.LabeledProfile{Profile: r, Labels: flushed.Labels}
			p.trySaveProfile(t.Context(), result)
			require.Equal(t, test.stored, listener)
			require.Len(t, storage.Profiles, 1)

			p.storage = &failingProfileStorage{storage}
			p.trySaveProfile(t.Context(), result)
			require.Equal(t, test.stored, listener)
			require.Len(t, storage.Profiles, 1)

			p.storage = storage
			empty := b.RestartProfiles()
			require.Len(t, empty.Profiles, 1)
			require.Zero(t, empty.Profiles[0].Meta.SampleCount)
			p.trySaveProfile(t.Context(), client.LabeledProfile{Profile: empty.Profiles[0]})
			require.Len(t, storage.Profiles, 1)
			require.Equal(t, test.stored, listener)
		})
	}
}
