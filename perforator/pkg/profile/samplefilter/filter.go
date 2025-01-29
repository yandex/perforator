package samplefilter

import (
	pprof "github.com/google/pprof/profile"

	"github.com/yandex/perforator/perforator/pkg/foreach"
)

type SampleFilter interface {
	Matches(labels map[string][]string) bool
}

func FilterProfilesBySampleFilters(profiles []*pprof.Profile, filters ...SampleFilter) (res []*pprof.Profile) {
	return foreach.Map(profiles, func(p *pprof.Profile) *pprof.Profile {
		p.Sample = foreach.Filter(p.Sample, func(sample *pprof.Sample) bool {
			ok := true
			for _, filter := range filters {
				if !filter.Matches(sample.Label) {
					ok = false
					break
				}
			}
			return ok
		})
		return p
	})
}
