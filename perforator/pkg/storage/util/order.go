package util

import (
	"github.com/yandex/perforator/perforator/proto/perforator"
)

type SortOrder struct {
	Columns    []string
	Descending bool
}

func SortOrderFromServicesProto(order *perforator.ListServicesOrderByClause) SortOrder {
	if order == nil {
		return SortOrder{Columns: []string{"service"}}
	}

	switch *order {
	case perforator.ListServicesOrderByClause_Services:
		return SortOrder{Columns: []string{"service"}}
	case perforator.ListServicesOrderByClause_ProfileCount:
		return SortOrder{Columns: []string{"profile_count"}, Descending: true}
	default:
		return SortOrder{Columns: []string{"service"}}
	}
}

func SortOrderFromProto(order *perforator.SortOrder) SortOrder {
	return SortOrder{
		Columns:    order.GetColumns(),
		Descending: order.GetDirection() == perforator.SortOrder_Descending,
	}
}
