package nativepipeline

import (
	"context"
	"maps"
	"sort"
	"sync/atomic"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/process"
	"github.com/yandex/perforator/perforator/pkg/env"
	"github.com/yandex/perforator/perforator/pkg/linux"
)

// ProcessPublisher implements process.MetadataPublisher for State.
// Versions increase across publications, removals and PID reuse.
type ProcessPublisher struct {
	state        *State
	log          log.Logger
	envWhitelist map[string]struct{}
	version      atomic.Uint64
}

// NewProcessPublisher copies the whitelist. Use one publisher for the lifetime
// of State; State must outlive its callbacks.
func NewProcessPublisher(
	state *State,
	l log.Logger,
	envWhitelist map[string]struct{},
) *ProcessPublisher {
	return &ProcessPublisher{
		state:        state,
		log:          l,
		envWhitelist: maps.Clone(envWhitelist),
	}
}

func (p *ProcessPublisher) PublishProcess(_ context.Context, info process.MetadataSnapshot) error {
	version := p.version.Add(1)
	snapshot := ProcessSnapshot{
		Key:      info.Key,
		Version:  version,
		Mappings: make([]Mapping, 0, len(info.Mappings)),
	}
	for _, m := range info.Mappings {
		snapshot.Mappings = append(snapshot.Mappings, Mapping{
			Begin:      m.Begin,
			End:        m.End,
			Offset:     m.EffectiveOffset,
			FileOffset: m.FileOffset,
			Path:       m.Path,
			BuildID:    m.BuildID,
		})
	}
	snapshot.TLSNames = make([]TLSName, 0, len(info.TLSNames))
	for offset, name := range info.TLSNames {
		snapshot.TLSNames = append(snapshot.TLSNames, TLSName{Offset: offset, Name: name})
	}
	sort.Slice(snapshot.TLSNames, func(i, j int) bool {
		return snapshot.TLSNames[i].Offset < snapshot.TLSNames[j].Offset
	})

	snapshot.Environment = make([]StringLabel, 0, len(p.envWhitelist))
	for key := range p.envWhitelist {
		if value, ok := info.Environment[key]; ok {
			snapshot.Environment = append(snapshot.Environment, StringLabel{
				Key:   env.BuildEnvLabelKey(key),
				Value: value,
			})
		}
	}
	sort.Slice(snapshot.Environment, func(i, j int) bool {
		return snapshot.Environment[i].Key < snapshot.Environment[j].Key
	})

	return p.state.ReplaceProcess(&snapshot)
}

func (p *ProcessPublisher) OnProcessDeath(
	_ context.Context,
	key linux.ProcessKey,
) {
	version := p.version.Add(1)
	if err := p.state.RemoveProcess(key, version); err != nil && p.log != nil {
		p.log.Error("Failed to remove native process snapshot",
			log.UInt32("pid", uint32(key.Pid)),
			log.UInt64("start_time", key.ProcessStartTime),
			log.Error(err),
		)
	}
}

var _ process.MetadataPublisher = (*ProcessPublisher)(nil)
