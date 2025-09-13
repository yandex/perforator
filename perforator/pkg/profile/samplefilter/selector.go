package samplefilter

import (
	"github.com/yandex/perforator/observability/lib/querylang"
)

func ExtractSelectorFilters(selector *querylang.Selector) ([]SampleFilter, error) {
	var filters []SampleFilter

	tlsFilter, err := BuildTLSFilter(selector)
	if err != nil {
		return nil, err
	}
	filters = append(filters, tlsFilter)

	envFilter, envErr := BuildEnvFilter(selector)
	if envErr != nil {
		return nil, envErr
	}
	filters = append(filters, envFilter)

	buildIDFilter, err := BuildBuildIDFilter(selector)
	if err != nil {
		return nil, err
	}
	filters = append(filters, buildIDFilter)

	return filters, nil
}
