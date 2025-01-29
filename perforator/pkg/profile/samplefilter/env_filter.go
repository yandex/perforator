package samplefilter

import (
	"fmt"

	"github.com/yandex/perforator/observability/lib/querylang"
	"github.com/yandex/perforator/perforator/pkg/env"
	"github.com/yandex/perforator/perforator/pkg/profilequerylang"
)

type envFilter map[string]string

func (ef envFilter) Matches(labels map[string][]string) bool {
	// Store set in order to solve case with duplicate label keys.
	matches := make(map[string]struct{})
	for k, v := range labels {
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
