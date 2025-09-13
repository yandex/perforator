package samplefilter

import (
	"github.com/yandex/perforator/observability/lib/querylang"
	profilepb "github.com/yandex/perforator/perforator/proto/profile"
)

func FillProtoSampleFilter(selector *querylang.Selector, proto *profilepb.SampleFilter) error {
	filters, err := ExtractSelectorFilters(selector)
	if err != nil {
		return err
	}

	for _, filter := range filters {
		filter.AppendToProto(proto)
	}

	return nil
}
