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

type totalSelfCyclesOptions struct {
	function string
}

type CountTotalSelfCyclesOption func(*totalSelfCyclesOptions)

func WithFunctionFilter(function string) CountTotalSelfCyclesOption {
	return func(options *totalSelfCyclesOptions) {
		options.function = function
	}
}

type AggregationStorage interface {
	SaveClusterTopEntry(ctx context.Context, servicePerfTop *ServicePerfTop) error
	AggregateClusterTop(ctx context.Context, generation uint32, filter *Filter, aggregationType GroupByMode, pagination util.Pagination, sortOrder SortOrder) ([]*AggregationValue, error)
	CountTotalSelfCycles(ctx context.Context, generation uint32, options ...CountTotalSelfCyclesOption) (*big.Int, error)
	CountTotalCumulativeCycles(ctx context.Context, generation uint32, funcFilter string) (*big.Int, error)
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
