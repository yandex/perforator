package aggregated

import (
	"context"
	"math/big"

	"github.com/yandex/perforator/perforator/pkg/storage/util"
)

type GroupByMode string

const (
	GroupByFunction GroupByMode = "function"
	GroupByService  GroupByMode = "service"
)

type MatchMode string

const (
	ExactMatch     MatchMode = "exact"
	RegexMatch     MatchMode = "regex"
	SubstringMatch MatchMode = "substr"
)

type Filter struct {
	FunctionFilter string
	ServiceFilter  string
	// Controls FunctionFilter mode
	// for ServiceFilter always exact match
	FunctionFilterMatchMode MatchMode
}

type SortOrder int32

const (
	SelfTimeSortOrder SortOrder = iota
	CumulativeTimeSortOrder
)

type AggregationStorage interface {
	SaveClusterTopEntry(ctx context.Context, servicePerfTop *ServicePerfTop) error
	AggregateClusterTop(ctx context.Context, generation uint32, filter *Filter, aggregationType GroupByMode, pagination util.Pagination, sortOrder SortOrder) ([]*AggregationValue, error)
	CountTotalCycles(ctx context.Context, generation uint32, funcFilter string) (*TotalCycles, error)
}

type Function struct {
	Name             string
	SelfCycles       big.Int
	CumulativeCycles big.Int
}

type ServicePerfTop struct {
	Generation  int
	ServiceName string

	Functions []Function
}
