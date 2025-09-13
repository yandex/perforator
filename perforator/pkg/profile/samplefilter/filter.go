package samplefilter

import (
	pprof "github.com/google/pprof/profile"

	"github.com/yandex/perforator/perforator/pkg/foreach"
	profilepb "github.com/yandex/perforator/perforator/proto/profile"
)

type SampleFilter interface {
	Matches(sample *pprof.Sample) bool
	AppendToProto(filter *profilepb.SampleFilter)
}

func FilterProfilesBySampleFilters(profiles []*pprof.Profile, filters ...SampleFilter) (res []*pprof.Profile) {
	return foreach.Map(profiles, func(p *pprof.Profile) *pprof.Profile {
		p.Sample = foreach.Filter(p.Sample, func(sample *pprof.Sample) bool {
			ok := true
			for _, filter := range filters {
				if !filter.Matches(sample) {
					ok = false
					break
				}
			}
			return ok
		})
		return p
	})
}
