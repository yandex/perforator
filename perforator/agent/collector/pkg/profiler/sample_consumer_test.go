package profiler

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/yandex/perforator/library/go/core/metrics/nop"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/profile"
	"github.com/yandex/perforator/perforator/internal/unwinder"
	"github.com/yandex/perforator/perforator/pkg/linux/clock"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

func TestEnvWhitelist(t *testing.T) {
	var sample unwinder.RecordSample
	l := xlog.ForTest(t)
	clockConverter, _ := clock.NewMonotonicClockConverter(l.Logger(), nop.Registry{})
	sampleConsumer := newOneShotSampleConsumer(
		&Profiler{
			envWhitelist: map[string]struct{}{
				"key1": {},
				"key2": {},
			},
			clockConverter: clockConverter,
		},
		DefaultSampleConsumerFeatures(),
		nil,
		&guardedProfileBuilder{multiProfileBuilder: newMultiProfileBuilder(nil)},
		&sample,
	)

	processEnvs := map[string]string{"secret1": "value1", "key1": "value1"}
	sampleConsumer.doCollectEnvironment(processEnvs)

	builder := sampleConsumer.initBuilderMinimal("cpu", []profile.SampleType{{Kind: "cpu", Unit: "cycles"}})
	sampleConsumer.collectEnvironmentInto(builder)
	// Mark sample as nonzero to prevent compaction.
	builder.AddValue(1)
	// Add a dummy location to prevent sample being skipped in Finish.
	builder.AddNativeLocation(1).Finish()
	builder.Finish()

	profile := sampleConsumer.profileBuilder.RestartProfiles()
	require.NotEmpty(t, profile.Profiles)

	firstProfile := profile.Profiles[0]

	require.Equal(t, 1, len(firstProfile.Sample))
	writtenSample := firstProfile.Sample[0]
	require.Equal(t, 1, len(writtenSample.Label))
	require.Equal(t, map[string][]string{"env:key1": {"value1"}}, writtenSample.Label)
}

func TestNoEmptySamples(t *testing.T) {
	var sample unwinder.RecordSample
	l := xlog.ForTest(t)
	clockConverter, _ := clock.NewMonotonicClockConverter(l.Logger(), nop.Registry{})
	sampleConsumer := newOneShotSampleConsumer(
		&Profiler{
			envWhitelist: map[string]struct{}{
				"key1": {},
				"key2": {},
			},
			clockConverter: clockConverter,
		},
		DefaultSampleConsumerFeatures(),
		nil,
		&guardedProfileBuilder{multiProfileBuilder: newMultiProfileBuilder(nil)},
		&sample,
	)

	builder := sampleConsumer.initBuilderMinimal("cpu", []profile.SampleType{{Kind: "cpu", Unit: "cycles"}})
	// Mark sample as nonzero to prevent compaction.
	builder.AddValue(1)
	// No locations in builder -> sample shouldn't be added into profile.
	builder.Finish()

	profile := sampleConsumer.profileBuilder.RestartProfiles()
	require.Equal(t, 1, len(profile.Profiles))

	firstProfile := profile.Profiles[0]
	require.Equal(t, 0, len(firstProfile.Sample))
}
