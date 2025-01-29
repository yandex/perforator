package profiler

import (
	"time"

	"golang.org/x/exp/maps"

	"github.com/yandex/perforator/perforator/agent/collector/pkg/profile"
)

////////////////////////////////////////////////////////////////////////////////

type labeledAgentProfiles struct {
	Profiles []*profile.Profile
	Labels   map[string]string
}

type multiProfileBuilder struct {
	labels           map[string]string
	caches           *profile.DefaultMap[uint32, profile.ProcessCache]
	builders         map[string]*profile.Builder
	profileStartTime time.Time
}

func newMultiProfileBuilder(labels map[string]string) *multiProfileBuilder {
	builder := multiProfileBuilder{
		labels:   labels,
		caches:   profile.NewProcessCaches(),
		builders: make(map[string]*profile.Builder),
	}
	builder.StartNewProfiles()

	return &builder
}

func (b *multiProfileBuilder) StartNewProfiles() {
	b.profileStartTime = time.Now()
	for _, builder := range b.builders {
		builder.SetStartTime(b.profileStartTime)
	}
}

func (b *multiProfileBuilder) RestartProfiles() labeledAgentProfiles {
	now := time.Now()

	profiles := make([]*profile.Profile, 0, len(b.builders))
	for _, builder := range b.builders {
		builder.SetEndTime(now)
		profiles = append(profiles, builder.Finish())
	}
	b.caches.Clear()
	b.StartNewProfiles()

	result := labeledAgentProfiles{
		Profiles: profiles,
		Labels:   map[string]string{},
	}
	maps.Copy(result.Labels, b.labels)

	return result
}

func (b *multiProfileBuilder) EnsureBuilder(name string, sampleTypes []profile.SampleType) *profile.Builder {
	builder := b.builders[name]
	if builder != nil {
		return builder
	}

	builder = profile.NewBuilderWithCaches(b.caches)
	builder.SetStartTime(b.profileStartTime)
	for _, sampleType := range sampleTypes {
		builder.AddSampleType(sampleType.Kind, sampleType.Unit)
	}
	b.builders[name] = builder

	return builder
}

func (b *multiProfileBuilder) ProfileStartTime() time.Time {
	return b.profileStartTime
}

////////////////////////////////////////////////////////////////////////////////
