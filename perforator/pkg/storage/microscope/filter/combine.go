package filter

import "github.com/yandex/perforator/perforator/pkg/storage/profile/meta"

var (
	_ = Filter(&CombinedFilter{})
)

type CombinedFilter struct {
	filters []Filter
}

func NewCombinedFilter(filters []Filter) *CombinedFilter {
	return &CombinedFilter{
		filters: filters,
	}
}

func (f *CombinedFilter) Filter(meta *meta.ProfileMetadata) bool {
	for _, filter := range f.filters {
		if filter.Filter(meta) {
			return true
		}
	}

	return false
}
