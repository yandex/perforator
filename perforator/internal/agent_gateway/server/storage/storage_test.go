package storage

import (
	"context"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/yandex/perforator/perforator/internal/xmetrics"
	"github.com/yandex/perforator/perforator/pkg/profilequerylang"
	"github.com/yandex/perforator/perforator/pkg/sampletype"
	profilemeta "github.com/yandex/perforator/perforator/pkg/storage/profile/meta"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

func newSamplingService(moduloByEvent map[string]uint64) *Service {
	reg := xmetrics.NewRegistry()

	service := &Service{
		opts: &options{
			samplingModulo:        1,
			samplingModuloByEvent: moduloByEvent,
		},
		metrics: &serviceMetrics{
			droppedProfiles:     reg.WithTags(map[string]string{"kind": "dropped"}).Counter("profiles.count"),
			sampledProfiles:     reg.WithTags(map[string]string{"kind": "sampled"}).Counter("profiles.count"),
			microscopedProfiles: reg.WithTags(map[string]string{"kind": "microscoped"}).Counter("profiles.count"),
		},
		profileSamplerByTypes: make(map[string]*moduloSampler),
	}

	return service
}

func admitMetas(service *Service, l xlog.Logger, eventTypes []string) []*profilemeta.ProfileMetadata {
	metas := createMetasWithEventType(
		&profilemeta.ProfileMetadata{Attributes: make(map[string]string)},
		eventTypes,
	)

	return service.sampleProfiles(context.Background(), l, metas)
}

func admitEventTypes(service *Service, l xlog.Logger, eventTypes []string) []string {
	admitted := make([]string, 0, len(eventTypes))
	for _, meta := range admitMetas(service, l, eventTypes) {
		admitted = append(admitted, meta.MainEventType)
	}

	return admitted
}

func TestSampleProfilesAdmitsAllEventTypesTogether(t *testing.T) {
	const modulo = 30

	cpu := sampletype.SampleTypeCPUCycles
	wall := sampletype.SampleTypeWallSeconds

	service := newSamplingService(map[string]uint64{cpu: modulo, wall: modulo})
	l := xlog.NewNop()

	const profiles = 10 * modulo

	stored := 0
	for i := 0; i < profiles; i++ {
		admitted := admitEventTypes(service, l, []string{cpu, wall})

		require.NotEqual(t, 1, len(admitted),
			"event types of one profile must be admitted together, otherwise they land on different blobs")

		if len(admitted) > 0 {
			require.ElementsMatch(t, []string{cpu, wall}, admitted)
			stored++
		}
	}

	require.Equal(t, profiles/modulo, stored)
}

func TestSampleProfilesUsesMostFrequentRate(t *testing.T) {
	const wallModulo = 30

	signal := sampletype.SampleTypeSignalCount
	wall := sampletype.SampleTypeWallSeconds

	service := newSamplingService(map[string]uint64{signal: 1, wall: wallModulo})
	l := xlog.NewNop()

	const profiles = 10 * wallModulo

	signals, walls := 0, 0
	for i := 0; i < profiles; i++ {
		for _, eventType := range admitEventTypes(service, l, []string{signal, wall}) {
			switch eventType {
			case signal:
				signals++
			case wall:
				walls++
			}
		}
	}

	require.Equal(t, profiles, signals, "the most frequent type keeps the whole profile admitted")
	require.Equal(t, profiles, walls, "a rarely sampled type is dragged along by the frequent one, never dropped separately")
}

func TestSampleProfilesKeepsSetsIndependent(t *testing.T) {
	const modulo = 30

	cpu := sampletype.SampleTypeCPUCycles
	wall := sampletype.SampleTypeWallSeconds

	service := newSamplingService(map[string]uint64{wall: modulo})
	l := xlog.NewNop()

	const profiles = 10 * modulo

	cpus, walls := 0, 0
	for i := 0; i < profiles; i++ {
		for _, eventType := range admitEventTypes(service, l, []string{cpu}) {
			if eventType == cpu {
				cpus++
			}
		}
		for _, eventType := range admitEventTypes(service, l, []string{wall}) {
			if eventType == wall {
				walls++
			}
		}
	}

	require.Equal(t, profiles, cpus, "a set sampled at modulo 1 must always be admitted")
	require.Equal(t, profiles/modulo, walls, "a rarely sampled set must keep its own rate")
}

func TestSampleProfilesSamplesAtMinimumModulo(t *testing.T) {
	const (
		cpuModulo  = 10
		wallModulo = 30
		profiles   = 30000
	)

	cpu := sampletype.SampleTypeCPUCycles
	wall := sampletype.SampleTypeWallSeconds

	service := newSamplingService(map[string]uint64{cpu: cpuModulo, wall: wallModulo})
	l := xlog.NewNop()

	stored := 0
	for i := 0; i < profiles; i++ {
		if len(admitEventTypes(service, l, []string{cpu, wall})) > 0 {
			stored++
		}
	}

	require.Equal(t, profiles/cpuModulo, stored)
}

func TestSampleProfilesTreatsTypeOrderAsSameSet(t *testing.T) {
	const modulo = 30

	cpu := sampletype.SampleTypeCPUCycles
	wall := sampletype.SampleTypeWallSeconds

	service := newSamplingService(map[string]uint64{cpu: modulo, wall: modulo})
	l := xlog.NewNop()

	const profiles = 10 * modulo

	stored := 0
	for i := 0; i < profiles; i++ {
		// Alternate the order of the same set; both must share one sampler.
		eventTypes := []string{cpu, wall}
		if i%2 == 1 {
			eventTypes = []string{wall, cpu}
		}
		if len(admitEventTypes(service, l, eventTypes)) > 0 {
			stored++
		}
	}

	require.Equal(t, profiles/modulo, stored,
		"permutations of the same type set must be counted by a single sampler")
}

func TestSampleProfilesSharesWeightAcrossEventTypes(t *testing.T) {
	const wallModulo = 30

	signal := sampletype.SampleTypeSignalCount
	wall := sampletype.SampleTypeWallSeconds

	service := newSamplingService(map[string]uint64{signal: 1, wall: wallModulo})
	l := xlog.NewNop()

	admitted := admitMetas(service, l, []string{signal, wall})
	require.Len(t, admitted, 2, "the whole profile is sampled at the minimum modulo 1, so both types are stored")

	weights := map[string]string{}
	for _, meta := range admitted {
		weights[meta.MainEventType] = meta.Attributes[profilequerylang.WeightLabel]
	}

	require.Equal(t, strconv.Itoa(1), weights[signal])
	require.Equal(t, strconv.Itoa(1), weights[wall],
		"all metas of a profile carry the same weight equal to the minimum modulo")
}
