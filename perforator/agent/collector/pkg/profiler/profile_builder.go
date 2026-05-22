package profiler

import (
	"bytes"
	"fmt"
	"sync"
	"time"

	pprof "github.com/google/pprof/profile"
	"golang.org/x/exp/maps"

	"github.com/yandex/perforator/perforator/agent/collector/pkg/profile"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/profileformat"
	"github.com/yandex/perforator/perforator/pkg/cprofile"
	"github.com/yandex/perforator/perforator/pkg/profile/bundle"
)

////////////////////////////////////////////////////////////////////////////////

type labeledAgentProfiles struct {
	Profiles []*profile.Profile
	Labels   map[string]string
}

////////////////////////////////////////////////////////////////////////////////

type multiProfileBuilder struct {
	mu               sync.RWMutex
	labels           map[string]string
	caches           *profile.DefaultMap[uint32, profile.ProcessCache]
	builders         map[string]*profile.Builder
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
		builders:      make(map[string]*profile.Builder),
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

	profiles := make([]*profile.Profile, 0, len(b.builders))
	for _, builder := range b.builders {
		switch b.profileFormat {
		case profileformat.Yaprof:
			profiles = append(profiles, buildYaprofAgentProfile(builder.FinishRaw(), b.labels))
		default:
			profiles = append(profiles, buildPprofAgentProfile(builder.Finish(), b.labels))
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

func buildPprofAgentProfile(compactedProfile *profile.Profile, labels map[string]string) *profile.Profile {
	addProfileComments(compactedProfile, labels)

	profileBytes := bytes.NewBuffer(nil)
	if err := compactedProfile.WriteUncompressed(profileBytes); err != nil {
		panic(fmt.Errorf("failed to serialize pprof profile: %w", err))
	}

	compactedProfile.Bundle = bundle.NewPprofBundle(profileBytes.Bytes())
	return compactedProfile
}

func buildYaprofAgentProfile(rawProfile *profile.Profile, labels map[string]string) *profile.Profile {
	addProfileComments(rawProfile, labels)
	rawProfile.PeriodType = &pprof.ValueType{}

	profileBytes := bytes.NewBuffer(nil)
	if err := rawProfile.WriteUncompressed(profileBytes); err != nil {
		panic(fmt.Errorf("failed to serialize raw pprof profile: %w", err))
	}

	yaprofBytes, err := cprofile.PprofToYaprof(profileBytes.Bytes())
	if err != nil {
		panic(fmt.Errorf("failed to convert pprof to yaprof: %w", err))
	}

	rawProfile.Bundle = bundle.NewYaprofBundle(yaprofBytes)
	return rawProfile
}

func addProfileComments(profile *profile.Profile, labels map[string]string) {
	for k, v := range labels {
		profile.Comments = append(profile.Comments, fmt.Sprintf("%s:%s", k, v))
	}
}

func (b *multiProfileBuilder) EnsureBuilder(name string, sampleTypes []profile.SampleType) *profile.Builder {
	b.mu.Lock()
	defer b.mu.Unlock()
	existing := b.builders[name]
	if existing != nil {
		return existing
	}

	builder := profile.NewBuilderWithCaches(b.caches)
	builder.SetStartTime(b.profileStartTime)
	for _, sampleType := range sampleTypes {
		builder.AddSampleType(sampleType.Kind, sampleType.Unit)
	}
	b.builders[name] = builder

	return builder
}

func (b *multiProfileBuilder) ProfileStartTime() time.Time {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.profileStartTime
}

////////////////////////////////////////////////////////////////////////////////
