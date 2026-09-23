package profile

import (
	"bytes"
	"errors"
	"slices"
	"time"

	"github.com/yandex/perforator/perforator/agent/collector/pkg/profileformat"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/profileresult"
	"github.com/yandex/perforator/perforator/pkg/env"
	"github.com/yandex/perforator/perforator/pkg/profile/bundle"
	"github.com/yandex/perforator/perforator/pkg/sampletype"
)

// ToResult adapts a completed Go builder profile without retaining its mutable model.
// Counts preserve the Go sender's semantics: one notification per finished builder
// sample, even if conversion to yaprof subsequently combines identical samples.
func (p *Profile) ToResult(format profileformat.ProfileFormat) (*profileresult.Result, error) {
	if p == nil || p.Profile == nil {
		return nil, errors.New("nil Go profile")
	}
	if err := p.CheckValid(); err != nil {
		return nil, err
	}
	var body bytes.Buffer
	if err := p.WriteUncompressed(&body); err != nil {
		return nil, err
	}
	b := bundle.NewPprofBundle(body.Bytes())
	switch format {
	case profileformat.Pprof:
	case profileformat.Yaprof:
		_, err := b.GetOrConvertYaprof()
		if err != nil {
			return nil, err
		}
		// Keep the original pprof bytes for Go consumers (local storage and CLI).
		// A yaprof round trip can aggregate samples and normalize numeric units.
	default:
		return nil, errors.New("unsupported profile format: " + string(format))
	}
	meta := profileresult.Meta{
		BuildIDs:        getProfileBuildIDs(p),
		Envs:            getProfileEnvs(p),
		EventTypes:      getProfileEventTypes(p),
		SignalTypes:     getProfileSignalTypes(p),
		PIDSampleCounts: make(map[int64]int),
		SampleCount:     len(p.Sample),
	}
	if p.TimeNanos != 0 {
		meta.StartTimestamp = time.Unix(0, p.TimeNanos)
		meta.EndTimestamp = meta.StartTimestamp.Add(time.Duration(p.DurationNanos))
	}
	for _, s := range p.Sample {
		if pids := s.NumLabel["pid"]; len(pids) == 1 {
			meta.PIDSampleCounts[pids[0]]++
		}
	}
	return &profileresult.Result{Bundle: b, Meta: meta}, nil
}

func getProfileBuildIDs(profile *Profile) []string {
	ids := make([]string, 0, len(profile.Mapping))
	known := make(map[string]bool)
	for _, m := range profile.Mapping {
		if m == nil || m.BuildID == "" {
			continue
		}

		id := m.BuildID
		if known[id] {
			continue
		}

		known[id] = true
		ids = append(ids, id)
	}
	slices.Sort(ids)
	return ids
}

func getProfileEnvs(profile *Profile) []string {
	res := make([]string, 0)
	seenEnvs := make(map[string]struct{})
	for _, s := range profile.Sample {
		if s == nil {
			continue
		}
		for key, values := range s.Label {
			if len(values) == 0 {
				continue
			}
			value := values[0]
			if envKey, ok := env.BuildEnvKeyFromLabelKey(key); ok {
				concatenatedEnv := env.BuildConcatenatedEnv(envKey, value)
				if _, seen := seenEnvs[concatenatedEnv]; !seen {
					seenEnvs[concatenatedEnv] = struct{}{}
					res = append(res, concatenatedEnv)
				}
			}
		}
	}
	return res
}

func getProfileEventTypes(profile *Profile) []string {
	if len(profile.SampleType) == 0 {
		return []string{sampletype.SampleTypeCPUCycles}
	}

	res := make([]string, 0, len(profile.SampleType))
	for _, sampleType := range profile.SampleType {
		res = append(res, sampletype.SampleTypeToString(sampleType))
	}

	return res
}

// getProfileSignalTypes returns a slice of unique signal names that were present in a profile.
func getProfileSignalTypes(profile *Profile) []string {
	signalSet := make(map[string]struct{})

	for _, sample := range profile.Sample {
		if values, exists := sample.Label["signal:name"]; exists {
			for _, value := range values {
				if value != "" {
					signalSet[value] = struct{}{}
				}
			}
		}
	}

	result := make([]string, 0, len(signalSet))
	for signal := range signalSet {
		result = append(result, signal)
	}
	slices.Sort(result)
	return result
}
