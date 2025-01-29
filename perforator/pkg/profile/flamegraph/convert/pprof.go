package convert

import (
	"fmt"

	"github.com/google/pprof/profile"

	"github.com/yandex/perforator/library/go/slices"
	"github.com/yandex/perforator/perforator/pkg/profile/flamegraph/collapsed"
)

func PProfToCollapsed(prof *profile.Profile) (*collapsed.Profile, error) {
	sampleTypeIdx := 0
	for i, value := range prof.SampleType {
		if value.Type == prof.DefaultSampleType {
			sampleTypeIdx = i
			break
		}
	}
	res := &collapsed.Profile{
		Samples: make([]collapsed.Sample, len(prof.Sample)),
	}
	for i := range prof.Sample {
		sample := &res.Samples[i]
		sample.Value = prof.Sample[i].Value[sampleTypeIdx]
		sample.Stack = make([]string, 0, len(prof.Sample[i].Location))
		for _, loc := range prof.Sample[i].Location {
			for j := len(loc.Line) - 1; j >= 0; j-- {
				line := loc.Line[j]
				name := ""
				if line.Function.Name != "" {
					name = line.Function.Name
				} else if line.Function.SystemName != "" {
					name = line.Function.SystemName
				}
				if j != 0 {
					name += " (inlined)"
				}
				sample.Stack = append(sample.Stack, name)
			}

			if len(loc.Line) == 0 {
				name := ""
				if loc.Mapping == nil {
					name = fmt.Sprintf("0x%x", loc.Address)
				} else {
					name = fmt.Sprintf("0x%x @%s", loc.Address, loc.Mapping.File)
				}
				sample.Stack = append(sample.Stack, name)
			}
		}
		slices.Reverse(sample.Stack)
	}
	return res, nil
}

func CollapsedToPProf(prof *collapsed.Profile) (*profile.Profile, error) {
	res := &profile.Profile{
		SampleType: []*profile.ValueType{{
			Type: "event",
			Unit: "count",
		}},
		Sample: make([]*profile.Sample, len(prof.Samples)),
	}

	locations := make(map[string]*profile.Location)
	for i := range prof.Samples {
		res.Sample[i] = &profile.Sample{
			Value: []int64{prof.Samples[i].Value},
		}
		for _, function := range prof.Samples[i].Stack {
			loc, found := locations[function]
			if !found {
				funcPtr := &profile.Function{
					ID:   1 + uint64(len(res.Function)),
					Name: function,
				}
				loc = &profile.Location{
					ID: 1 + uint64(len(res.Location)),
					Line: []profile.Line{{
						Function: funcPtr,
					}},
				}
				res.Function = append(res.Function, funcPtr)
				res.Location = append(res.Location, loc)
			}
			res.Sample[i].Location = append(res.Sample[i].Location, loc)
		}
		slices.Reverse(res.Sample[i].Location)
	}

	return res, nil
}
