package filter

import (
	"github.com/yandex/perforator/observability/lib/querylang"
	"github.com/yandex/perforator/observability/lib/querylang/operator"
	"github.com/yandex/perforator/perforator/pkg/profilequerylang"
)

var (
	labelToMicroscopeType = map[string]MicroscopeType{
		profilequerylang.PodIDLabel:   PodFilter,
		profilequerylang.NodeIDLabel:  NodeFilter,
		profilequerylang.ServiceLabel: ServiceFilter,
	}
)

func DeductMicroscopeType(selector *querylang.Selector) MicroscopeType {
	var notTSMatcher *querylang.Matcher
	for _, matcher := range selector.Matchers {
		if matcher.Field == profilequerylang.TimestampLabel {
			continue
		}

		if notTSMatcher != nil {
			return AbstractFilter
		}

		notTSMatcher = matcher
	}

	if notTSMatcher == nil {
		return AbstractFilter
	}

	if _, ok := labelToMicroscopeType[notTSMatcher.Field]; !ok {
		return AbstractFilter
	}

	if len(notTSMatcher.Conditions) != 1 {
		return AbstractFilter
	}

	condition := notTSMatcher.Conditions[0]
	if condition.Operator != operator.Eq || condition.Inverse {
		return AbstractFilter
	}

	return labelToMicroscopeType[notTSMatcher.Field]
}

type DeductResult struct {
	Type  MicroscopeType
	Value string
}

func DeductMicroscope(selector *querylang.Selector) *DeductResult {
	tp := DeductMicroscopeType(selector)

	if tp == AbstractFilter {
		return &DeductResult{tp, ""}
	}

	for _, matcher := range selector.Matchers {
		if matcher.Field == profilequerylang.TimestampLabel {
			continue
		}

		return &DeductResult{tp, profilequerylang.ValueRepr(matcher.Conditions[0].Value)}
	}

	return nil
}
