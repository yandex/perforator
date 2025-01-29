package samplefilter

import (
	"fmt"

	"github.com/yandex/perforator/observability/lib/querylang"
	"github.com/yandex/perforator/perforator/pkg/profilequerylang"
	"github.com/yandex/perforator/perforator/pkg/tls"
)

type tlsFilter map[string]string

func (tf tlsFilter) Matches(labels map[string][]string) bool {
	// Store set in order to solve case with duplicate label keys.
	matches := make(map[string]struct{})
	for k, v := range labels {
		if len(v) == 0 {
			continue
		}
		tlsKey, parsed := tls.BuildTLSKeyFromLabelKey(k)
		if !parsed {
			continue
		}
		expected, ok := tf[tlsKey]
		// In theory, profile labels can have more than one value. We rely only on the first one.
		if ok && v[0] == expected {
			matches[tlsKey] = struct{}{}
		}
	}
	return len(matches) == len(tf)
}

func BuildTLSFilter(selector *querylang.Selector) (SampleFilter, error) {
	res := make(map[string]string)
	for _, matcher := range selector.Matchers {
		if tls.IsTLSMatcherField(matcher.Field) {
			tlsKey, ok := tls.BuildTLSKeyFromMatcherField(matcher.Field)
			if !ok {
				return nil, fmt.Errorf("failed to build TLS filters: failed to build tls key from %s", matcher.Field)
			}
			val, err := profilequerylang.ExtractEqualityMatch(matcher)
			if err != nil {
				return nil, fmt.Errorf("failed to build TLS filters: %w", err)
			}
			res[tlsKey] = val
		}
	}
	return tlsFilter(res), nil
}
