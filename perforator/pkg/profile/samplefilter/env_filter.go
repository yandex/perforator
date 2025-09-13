package samplefilter

import (
	"fmt"

	pprof "github.com/google/pprof/profile"

	"github.com/yandex/perforator/observability/lib/querylang"
	"github.com/yandex/perforator/perforator/pkg/env"
	"github.com/yandex/perforator/perforator/pkg/profilequerylang"
	profilepb "github.com/yandex/perforator/perforator/proto/profile"
)

type envFilter map[string]string

func (ef envFilter) Matches(sample *pprof.Sample) bool {
	// Store set in order to solve case with duplicate label keys.
	matches := make(map[string]struct{})
	for k, v := range sample.Label {
		if len(v) == 0 {
			continue
		}
		envKey, parsed := env.BuildEnvKeyFromLabelKey(k)
		if !parsed {
			continue
		}
		expected, ok := ef[envKey]
		// In theory, profile labels can have more than one value. We rely only on first one.
		if ok && v[0] == expected {
			matches[envKey] = struct{}{}
		}
	}
	return len(matches) == len(ef)
}

func (ef envFilter) AppendToProto(filter *profilepb.SampleFilter) {
	if filter.RequiredAllOfStringLabels == nil {
		filter.RequiredAllOfStringLabels = make(map[string]string)
	}

	for k, v := range ef {
		filter.RequiredAllOfStringLabels[env.BuildEnvLabelKey(k)] = v
	}
}

func BuildEnvFilter(selector *querylang.Selector) (SampleFilter, error) {
	res := make(map[string]string)
	for _, matcher := range selector.Matchers {
		envKey, ok := env.BuildEnvKeyFromMatcherField(matcher.Field)
		if !ok {
			continue
		}
		val, err := profilequerylang.ExtractEqualityMatch(matcher)
		if err != nil {
			return nil, fmt.Errorf("failed to build env filters with env %v: %w", matcher.Field, err)
		}
		res[envKey] = val
	}
	return envFilter(res), nil
}
