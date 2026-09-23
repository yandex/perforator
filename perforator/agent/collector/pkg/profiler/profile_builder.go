package profiler

import (
	"fmt"
	"slices"
	"sync"
	"time"

	pprof "github.com/google/pprof/profile"
	"golang.org/x/exp/maps"

	"github.com/yandex/perforator/perforator/agent/collector/pkg/profile"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/profileformat"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/profileresult"
)

////////////////////////////////////////////////////////////////////////////////

type labeledAgentProfiles struct {
	Profiles []*profileresult.Result
	Labels   map[string]string
}

////////////////////////////////////////////////////////////////////////////////

// profileBuilderWithSampleTypes associates a builder with its ordered value types.
// Each sample value corresponds to the kind and unit at the same index.
type profileBuilderWithSampleTypes struct {
	builder     *profile.Builder
	sampleTypes []profile.SampleType
}

type multiProfileBuilder struct {
	mu               sync.RWMutex
	labels           map[string]string
	caches           *profile.DefaultMap[uint32, profile.ProcessCache]
	builders         map[string][]profileBuilderWithSampleTypes
	profileStartTime time.Time
	profileFormat    profileformat.ProfileFormat
}

func newMultiProfileBuilder(labels map[string]string, pf profileformat.ProfileFormat) *multiProfileBuilder {
	if pf == "" {
		pf = profileformat.Pprof
	}

	builder := multiProfileBuilder{
		labels:        labels,
		caches:        profile.NewProcessCaches(),
		builders:      make(map[string][]profileBuilderWithSampleTypes),
		profileFormat: pf,
	}
	builder.startNewProfiles()

	return &builder
}

func (b *multiProfileBuilder) startNewProfiles() {
	b.profileStartTime = time.Now()
}

func (b *multiProfileBuilder) RestartProfiles() labeledAgentProfiles {
	b.mu.Lock()
	defer b.mu.Unlock()

	profiles := make([]*profileresult.Result, 0, len(b.builders))
	for _, builders := range b.builders {
		for _, entry := range builders {
			builder := entry.builder
			switch b.profileFormat {
			case profileformat.Yaprof:
				profiles = append(profiles, buildYaprofAgentProfile(builder.FinishRaw(), b.labels))
			default:
				profiles = append(profiles, buildPprofAgentProfile(builder.Finish(), b.labels))
			}
		}
	}
	b.caches.Clear()
	b.startNewProfiles()

	result := labeledAgentProfiles{
		Profiles: profiles,
		Labels:   map[string]string{},
	}
	maps.Copy(result.Labels, b.labels)

	return result
}

func buildPprofAgentProfile(p *profile.Profile, labels map[string]string) *profileresult.Result {
	return buildAgentProfile(p, labels, profileformat.Pprof)
}

func buildYaprofAgentProfile(p *profile.Profile, labels map[string]string) *profileresult.Result {
	p.PeriodType = &pprof.ValueType{}
	return buildAgentProfile(p, labels, profileformat.Yaprof)
}

func buildAgentProfile(p *profile.Profile, labels map[string]string, format profileformat.ProfileFormat) *profileresult.Result {
	addProfileComments(p, labels)
	result, err := p.ToResult(format)
	if err != nil {
		panic(fmt.Errorf("failed to serialize %s profile: %w", format, err))
	}
	return result
}

func addProfileComments(profile *profile.Profile, labels map[string]string) {
	for k, v := range labels {
		profile.Comments = append(profile.Comments, fmt.Sprintf("%s:%s", k, v))
	}
}

func (b *multiProfileBuilder) EnsureBuilder(name string, sampleTypes []profile.SampleType) *profile.Builder {
	b.mu.Lock()
	defer b.mu.Unlock()
	// Match sample kinds and units in order without allocating on the sample hot path.
	for _, entry := range b.builders[name] {
		if slices.Equal(entry.sampleTypes, sampleTypes) {
			return entry.builder
		}
	}

	builder := profile.NewBuilderWithCaches(b.caches)
	builder.SetStartTime(b.profileStartTime)
	for _, sampleType := range sampleTypes {
		builder.AddSampleType(sampleType.Kind, sampleType.Unit)
	}
	b.builders[name] = append(b.builders[name], profileBuilderWithSampleTypes{
		builder: builder, sampleTypes: slices.Clone(sampleTypes),
	})

	return builder
}

func (b *multiProfileBuilder) ProfileStartTime() time.Time {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.profileStartTime
}

////////////////////////////////////////////////////////////////////////////////
