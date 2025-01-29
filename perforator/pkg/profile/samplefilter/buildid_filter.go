package samplefilter

import (
	"fmt"
	"slices"

	"github.com/yandex/perforator/observability/lib/querylang"
	"github.com/yandex/perforator/perforator/pkg/profilequerylang"
)

type buildIDFilter string

const (
	nopBuildIDFilter    buildIDFilter = ""
	buildIDMatcherField string        = "buildid"
	buildIDLabelName    string        = "buildid"
)

func (bf buildIDFilter) Matches(labels map[string][]string) bool {
	if bf == nopBuildIDFilter {
		return true
	}
	actualBuildID, ok := labels[buildIDLabelName]
	if !ok {
		return false
	}
	return slices.Contains(actualBuildID, string(bf))
}

func BuildBuildIDFilter(selector *querylang.Selector) (SampleFilter, error) {
	for _, matcher := range selector.Matchers {
		if matcher.Field != buildIDMatcherField {
			continue
		}
		val, err := profilequerylang.ExtractEqualityMatch(matcher)
		if err != nil {
			return nil, fmt.Errorf("failed to extract desired build id: %w", err)
		}
		return buildIDFilter(val), nil
	}
	return nopBuildIDFilter, nil
}
