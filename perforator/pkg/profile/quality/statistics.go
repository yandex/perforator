package quality

import (
	"github.com/google/pprof/profile"

	"github.com/yandex/perforator/perforator/proto/perforator"
)

func CalculateProfileStatistics(profile *profile.Profile) *perforator.ProfileStatistics {
	stats := &perforator.ProfileStatistics{
		SampleValueSum:         make(map[string]float64),
		UniqueSampleCount:      uint64(len(profile.Sample)),
		TotalFrameCount:        0,
		UnmappedFrameCount:     0,
		UnsymbolizedFrameCount: 0,
		TotalBinaryCount:       0,
		UnavailableBinaryCount: 0,
	}

	referencedBinaries := make(map[string]struct{})
	symbolizedBinaries := make(map[string]struct{})

	types := make([]string, len(profile.SampleType))
	for i, typ := range profile.SampleType {
		types[i] = typ.Type + "." + typ.Unit
	}

	for _, sample := range profile.Sample {
		for i, value := range sample.Value {
			stats.SampleValueSum[types[i]] += float64(value)
		}

		stats.TotalFrameCount += uint64(len(sample.Location))
		for _, location := range sample.Location {
			symbolized := len(location.Line) > 0
			if !symbolized {
				stats.UnsymbolizedFrameCount++
			}

			if location.Mapping == nil {
				stats.UnmappedFrameCount++
				continue
			}

			referencedBinaries[location.Mapping.BuildID] = struct{}{}
			if len(location.Line) > 0 {
				symbolizedBinaries[location.Mapping.BuildID] = struct{}{}
			}
		}
	}

	stats.TotalBinaryCount = uint64(len(referencedBinaries))
	for id := range referencedBinaries {
		if _, ok := symbolizedBinaries[id]; !ok {
			stats.UnavailableBinaryCount++
		}
	}

	return stats
}
