package aggregated

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/Masterminds/squirrel"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/perforator/pkg/clickhouse"
	"github.com/yandex/perforator/perforator/pkg/foreach"
	"github.com/yandex/perforator/perforator/pkg/storage/util"
	"github.com/yandex/perforator/perforator/pkg/xlog"
	"github.com/yandex/perforator/perforator/proto/perforator"
)

type clusterTopRow struct {
	Generation       int     `ch:"generation"`
	Service          string  `ch:"service"`
	Function         string  `ch:"function"`
	SelfCycles       big.Int `ch:"self_cycles"`
	CumulativeCycles big.Int `ch:"cumulative_cycles"`
}

type ClickhouseAggregationStorage struct {
	l    xlog.Logger
	conn *clickhouse.Connection
}

type AggregationQuery struct {
	function string
	service  string
}

var (
	_ AggregationStorage = (*ClickhouseAggregationStorage)(nil)
)

func NewStorage(l xlog.Logger, conn *clickhouse.Connection) *ClickhouseAggregationStorage {
	l = l.WithName("clustertop_clickhouse")

	return &ClickhouseAggregationStorage{
		l:    l,
		conn: conn,
	}
}

type AggregationValue struct {
	Name                string  `ch:"name"`
	CpuCycles           big.Int `ch:"cpu_cycles"`
	CumulativeCpuCycles big.Int `ch:"sum_cumulative_cycles"`
}

const ESTIMATED_CPU_FREQ = 2.6 * 1_000_000_000
const PERFORATOR_SAMPLING_MODULO = 30
const INTERVAL_SEC = 3600

func fromCpuCyclesToCpuHours(cpuCycles *big.Int) float64 {
	nonSampledCycles := big.NewInt(PERFORATOR_SAMPLING_MODULO)
	nonSampledCycles = nonSampledCycles.Mul(cpuCycles, nonSampledCycles)
	cpuSeconds := nonSampledCycles.Div(nonSampledCycles, big.NewInt(ESTIMATED_CPU_FREQ))
	hours, _ := cpuSeconds.Div(cpuSeconds, big.NewInt(INTERVAL_SEC)).Float64()
	return hours
}

func scanTopRow(rows driver.Rows) (*AggregationValue, error) {
	var row AggregationValue
	if err := rows.ScanStruct(&row); err != nil {
		return nil, fmt.Errorf("failed to scan string from row: %w", err)

	}
	return &row, nil
}

func fromCpuCyclesToPercent(total *big.Int, current *big.Int) float64 {
	curr, _ := current.Float64()
	t, _ := total.Float64()
	return curr * 100 / t
}

func MapEntries(total *TotalCycles, entries []*AggregationValue) []*perforator.ClusterTopEntry {
	res := foreach.Map(entries, func(row *AggregationValue) *perforator.ClusterTopEntry {
		return &perforator.ClusterTopEntry{
			Name: row.Name,
			Count: &perforator.ClusterTopCount{
				Self:          fromCpuCyclesToCpuHours(&row.CpuCycles),
				Cumulative:    fromCpuCyclesToCpuHours(&row.CumulativeCpuCycles),
				SelfPct:       fromCpuCyclesToPercent(total.TotalSelfCycles, &row.CpuCycles),
				CumulativePct: fromCpuCyclesToPercent(total.TotalSelfCycles, &row.CumulativeCpuCycles),
			},
		}
	})
	return res
}

var groupByAggregation = map[GroupByMode]string{
	GroupByFunction: "function",
	GroupByService:  "service",
}

const DefaultPageSize = 100

func getComparisonOperator(mode MatchMode) string {
	switch mode {
	case ExactMatch:
		return "=="
	case RegexMatch:
		return "REGEXP"
	case SubstringMatch:
		return "LIKE"
	default:
		return ""
	}
}

const clusterTopTable = "cluster_top"

type TotalCycles struct {
	TotalSelfCycles *big.Int `ch:"total_self_cycles"`
}

func (s *ClickhouseAggregationStorage) CountTotalCycles(ctx context.Context, generation uint32, totalFunctionName string) (*TotalCycles, error) {
	builder := squirrel.Select("sum(self_cycles) as total_self_cycles").
		From(clusterTopTable).
		Where("generation = ?", generation)

	if totalFunctionName != "" {
		builder = builder.Where("function = ?", totalFunctionName)
	}

	sql, args, err := builder.ToSql()
	if err != nil {
		return nil, err
	}

	s.l.Debug(ctx, "Counting total cycles in clickhouse", log.String("sql", sql), log.Array("args", args))
	res, err := clickhouse.QueryWithRetries(s.l, ctx, s.conn, sql, func(r driver.Rows) (*TotalCycles, error) {
		var result TotalCycles
		if err := r.ScanStruct(&result); err != nil {
			return nil, fmt.Errorf("failed to scan from row: %w", err)
		}

		return &result, nil
	}, args...)
	if err != nil {
		return nil, err
	}

	if len(res) != 1 {
		return nil, errors.New("unexpected row count")
	}

	return res[0], nil
}

// aggregates cluster top based on
func (s *ClickhouseAggregationStorage) AggregateClusterTop(ctx context.Context, generation uint32, filter *Filter, aggregationType GroupByMode, pagination util.Pagination, sortOrder SortOrder) ([]*AggregationValue, error) {
	var sql string
	var err error

	groupBy := groupByAggregation[aggregationType]

	limit := pagination.Limit
	if limit == 0 {
		limit = DefaultPageSize
	}
	offset := pagination.Offset
	var orderByCycles string
	switch sortOrder {
	case SelfTimeSortOrder:
		orderByCycles = "cpu_cycles DESC"
	case CumulativeTimeSortOrder:
		orderByCycles = "sum_cumulative_cycles DESC"
	default:
		return nil, fmt.Errorf("unknown sort order")
	}

	builder := squirrel.
		Select(fmt.Sprintf("left(%s, 150) AS name, sum(self_cycles) AS cpu_cycles, sum(cumulative_cycles) as sum_cumulative_cycles", groupBy)).
		From(clusterTopTable).
		Where("generation = ?", generation).
		OrderBy(orderByCycles).
		Limit(limit).
		Offset(offset).
		GroupBy(groupBy)

	if filter != nil && filter.FunctionFilter != "" && filter.FunctionFilterMatchMode != "" {
		comparisonOperator := getComparisonOperator(filter.FunctionFilterMatchMode)
		searchValue := filter.FunctionFilter
		if filter.FunctionFilterMatchMode == SubstringMatch {
			searchValue = fmt.Sprintf("%%%s%%", filter.FunctionFilter)
		}
		builder = builder.
			Where(fmt.Sprintf("function %s ?", comparisonOperator), searchValue)
	}

	sql, args, err := builder.ToSql()
	if err != nil {
		return nil, err
	}

	s.l.Debug(ctx, "Aggregating cluster top data in clickhouse", log.String("sql", sql), log.Array("args", args))
	rows, err := clickhouse.QueryWithRetries(s.l, ctx, s.conn, sql, scanTopRow, args...)

	if err != nil {
		return nil, err
	}

	return rows, nil
}

const kMaxFunctionNameLength = 512

func (s *ClickhouseAggregationStorage) SaveClusterTopEntry(ctx context.Context, servicePerfTop *ServicePerfTop) error {

	batch, err := s.conn.PrepareBatch(
		ctx,
		"INSERT INTO cluster_top(generation, service, function, self_cycles, cumulative_cycles)",
	)
	if err != nil {
		return fmt.Errorf("failed to prepare clickhouse batch: %w", err)
	}

	defer func() { _ = batch.Abort() }()

	for _, function := range servicePerfTop.Functions {
		lengthLimitedFunctionName := function.Name
		if len(lengthLimitedFunctionName) > kMaxFunctionNameLength {
			lengthLimitedFunctionName = lengthLimitedFunctionName[:kMaxFunctionNameLength]
		}
		clickhouseRow := clusterTopRow{
			Generation:       servicePerfTop.Generation,
			Service:          servicePerfTop.ServiceName,
			Function:         lengthLimitedFunctionName,
			SelfCycles:       function.SelfCycles,
			CumulativeCycles: function.CumulativeCycles,
		}
		err := batch.AppendStruct(&clickhouseRow)
		if err != nil {
			return fmt.Errorf("failed to serialize clickhouse row: %w", err)
		}
	}

	return batch.Send()
}
