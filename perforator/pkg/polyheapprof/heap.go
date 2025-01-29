package polyheapprof

import (
	"bytes"
	"fmt"
	"os"
	"runtime/pprof"

	"github.com/google/pprof/profile"
)

func StartHeapProfileRecording() error {
	return startCHeapProfileRecording()
}

func ReadCurrentHeapProfile() (*profile.Profile, error) {
	var profiles []*profile.Profile

	goprof, err := readGoHeapProfile()
	if err != nil {
		return nil, fmt.Errorf("failed to read Go heap profile: %w", err)
	}
	profiles = append(profiles, goprof)

	cprof, err := readCHeapProfile()
	if err != nil {
		return nil, fmt.Errorf("failed to read C heap profile: %w", err)
	}
	profiles = append(profiles, cprof)

	for i, profile := range profiles {
		normalizeProfile(profile)
		for _, sampletype := range profile.SampleType {
			fmt.Fprintf(os.Stderr, "%d: %s.%s\n", i, sampletype.Type, sampletype.Unit)
		}
	}

	res, err := profile.Merge(profiles)
	if err != nil {
		return nil, fmt.Errorf("failed to merge C & Go heap profiles: %w", err)
	}

	return res, nil
}

func readGoHeapProfile() (*profile.Profile, error) {
	var buf bytes.Buffer

	err := pprof.WriteHeapProfile(&buf)
	if err != nil {
		return nil, err
	}

	return profile.ParseData(buf.Bytes())
}

var sampleTypeMapping = map[string]string{
	"inuse_objects": "inuse_objects",
	"allocations":   "inuse_objects",
	"inuse_space":   "inuse_space",
	"space":         "inuse_space",
}

func normalizeProfile(p *profile.Profile) *profile.Profile {
	var (
		sampleTypePosition = make([]int, len(p.SampleType))
		numSampleTypes     = 0
	)

	for i, sample := range p.SampleType {
		name, ok := sampleTypeMapping[sample.Type]

		if !ok {
			sampleTypePosition[i] = -1
			continue
		}

		p.SampleType[numSampleTypes] = &profile.ValueType{
			Type: name,
			Unit: sample.Unit,
		}
		sampleTypePosition[i] = numSampleTypes
		numSampleTypes++
	}

	p.SampleType = p.SampleType[:numSampleTypes]

	for _, sample := range p.Sample {
		values := make([]int64, numSampleTypes)
		for i, j := range sampleTypePosition {
			if j != -1 {
				values[j] = sample.Value[i]
			}
		}
		sample.Value = values
	}

	return p
}
