package aggregated

import (
	"context"
	"math/big"

	"github.com/yandex/perforator/perforator/pkg/storage/util"
	"github.com/yandex/perforator/perforator/proto/perforator"
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

type QueryRow struct {
	Function         string  `ch:"function"`
	Service          string  `ch:"service"`
	BuildID          string  `ch:"build_id"`
	CommitID         string  `ch:"commit_id"`
	BinaryPath       string  `ch:"binary_path"`
	Filename         string  `ch:"source_file"`
	SourceLine       uint32  `ch:"source_line"`
	SourceColumn     uint32  `ch:"source_column"`
	SelfCycles       big.Int `ch:"self_cycles"`
	CumulativeCycles big.Int `ch:"cumulative_cycles"`
}

type QueryResult struct {
	Rows    []*QueryRow
	HasMore bool
}

type AggregationStorage interface {
	QueryClusterTop(ctx context.Context, req *perforator.ClusterTopQueryRequest) (*QueryResult, error)
	SaveClusterTopEntry(ctx context.Context, result *JobResult) error
	AggregateClusterTop(ctx context.Context, generation uint32, filter *Filter, aggregationType GroupByMode, pagination util.Pagination, sortOrder SortOrder) ([]*AggregationValue, error)
	CountTotalSelfCycles(ctx context.Context, generation uint32, options ...CountTotalSelfCyclesOption) (*big.Int, error)
	CountTotalCumulativeCycles(ctx context.Context, generation uint32, funcFilter string) (*big.Int, error)
}

type Function struct {
	Name             string
	BuildID          string
	SourceFile       string
	SourceLine       uint32
	SourceColumn     uint32
	SelfCycles       big.Int
	CumulativeCycles big.Int
}

// JobResult contains one job's contribution to a generation's Cluster Top.
type JobResult struct {
	JobID       int64
	Generation  int
	BucketCount uint16
	ServiceName string

	Functions []Function
}
