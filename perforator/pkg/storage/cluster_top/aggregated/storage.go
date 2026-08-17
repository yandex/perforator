package aggregated

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	clickhousego "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/Masterminds/squirrel"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/perforator/pkg/clickhouse"
	"github.com/yandex/perforator/perforator/pkg/storage/util"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

type clusterTopRow struct {
	Generation       int     `ch:"generation"`
	Service          string  `ch:"service"`
	Function         string  `ch:"function"`
	SelfCycles       big.Int `ch:"self_cycles"`
	CumulativeCycles big.Int `ch:"cumulative_cycles"`
}

type ClickhouseAggregationStorage struct {
	l                 xlog.Logger
	conn              *clickhouse.Connection
	asyncInsertConfig AsyncInsertConfig
}

type AsyncInsertConfig struct {
	BusyTimeout    time.Duration `yaml:"busy_timeout"`
	MaxDataSize    uint64        `yaml:"max_data_size"`
	MaxQueryNumber uint64        `yaml:"max_query_number"`
}

type AggregationQuery struct {
	function string
	service  string
}

var (
	_ AggregationStorage = (*ClickhouseAggregationStorage)(nil)
)

func NewStorage(l xlog.Logger, conn *clickhouse.Connection, asyncInsertConfig AsyncInsertConfig) *ClickhouseAggregationStorage {
	l = l.WithName("clustertop_clickhouse")

	return &ClickhouseAggregationStorage{
		l:                 l,
		conn:              conn,
		asyncInsertConfig: asyncInsertConfig,
	}
}

type AggregationValue struct {
	Name                string  `ch:"name"`
	CpuCycles           big.Int `ch:"cpu_cycles"`
	CumulativeCpuCycles big.Int `ch:"sum_cumulative_cycles"`
}

func scanTopRow(rows driver.Rows) (*AggregationValue, error) {
	var row AggregationValue
	if err := rows.ScanStruct(&row); err != nil {
		return nil, fmt.Errorf("failed to scan string from row: %w", err)

	}
	return &row, nil
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

const (
	clusterTopTable                 = "cluster_top_v2"
	clusterTopByFunctionTable       = "cluster_top_by_function_v2"
	clusterTopGenerationTotalsTable = "cluster_top_generation_totals_v2"
)

func (s *ClickhouseAggregationStorage) CountTotalSelfCycles(ctx context.Context, generation uint32, options ...CountTotalSelfCyclesOption) (*big.Int, error) {
	optionsObject := &totalSelfCyclesOptions{}
	for _, option := range options {
		option(optionsObject)
	}

	var builder squirrel.SelectBuilder
	if optionsObject.function != "" {
		builder = squirrel.Select("sum(self_cycles) as total_self_cycles").
			From(clusterTopByFunctionTable).
			Where("generation = ?", generation).
			Where("function = ?", optionsObject.function)
	} else {
		builder = squirrel.Select("sum(total_self_cycles) as total_self_cycles").
			From(clusterTopGenerationTotalsTable).
			Where("generation = ?", generation)
	}

	return s.countTotal(ctx, builder)
}

func (s *ClickhouseAggregationStorage) CountTotalCumulativeCycles(ctx context.Context, generation uint32, totalFunctionName string) (*big.Int, error) {
	builder := squirrel.Select("sum(cumulative_cycles) as total_cumulative_cycles").
		From(clusterTopByFunctionTable).
		Where("generation = ?", generation)

	if totalFunctionName == "" {
		return nil, errors.New("total function name is empty")
	}

	builder = builder.Where("function = ?", totalFunctionName)

	return s.countTotal(ctx, builder)
}

func (s *ClickhouseAggregationStorage) countTotal(ctx context.Context, builder squirrel.SelectBuilder) (*big.Int, error) {
	sql, args, err := builder.ToSql()
	if err != nil {
		return nil, err
	}

	s.l.Debug(ctx, "Counting total cycles in clickhouse", log.String("sql", sql), log.Array("args", args))
	res, err := clickhouse.QueryWithRetries(s.l, ctx, s.conn, sql, func(r driver.Rows) (*big.Int, error) {
		var result big.Int
		if err := r.Scan(&result); err != nil {
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

	fromTable := clusterTopTable
	if aggregationType == GroupByFunction {
		fromTable = clusterTopByFunctionTable
	}
	// GroupByService with exact function filter reads clusterTopTable;
	// ClickHouse uses proj_by_function_service projection automatically.

	builder := squirrel.
		Select(fmt.Sprintf("%s AS name, sum(self_cycles) AS cpu_cycles, sum(cumulative_cycles) as sum_cumulative_cycles", groupBy)).
		From(fromTable).
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

func buildClusterTopRows(servicePerfTop *ServicePerfTop) []clusterTopRow {
	if servicePerfTop == nil || len(servicePerfTop.Functions) == 0 {
		return nil
	}

	rows := make([]clusterTopRow, 0, len(servicePerfTop.Functions))
	for _, function := range servicePerfTop.Functions {
		functionName := function.Name
		if len(functionName) > kMaxFunctionNameLength {
			functionName = functionName[:kMaxFunctionNameLength]
		}

		rows = append(rows, clusterTopRow{
			Generation:       servicePerfTop.Generation,
			Service:          servicePerfTop.ServiceName,
			Function:         functionName,
			SelfCycles:       function.SelfCycles,
			CumulativeCycles: function.CumulativeCycles,
		})
	}
	return rows
}

func (s *ClickhouseAggregationStorage) SaveClusterTopEntry(ctx context.Context, servicePerfTop *ServicePerfTop) error {
	rows := buildClusterTopRows(servicePerfTop)
	if len(rows) == 0 {
		return nil
	}

	const valuesPlaceholder = "(?, ?, ?, ?, ?)"
	var query strings.Builder
	query.Grow(len(rows) * (len(valuesPlaceholder) + 2))
	args := make([]any, 0, len(rows)*5)
	for i := range rows {
		if i > 0 {
			query.WriteString(", ")
		}
		query.WriteString(valuesPlaceholder)
		args = append(args,
			rows[i].Generation,
			rows[i].Service,
			rows[i].Function,
			&rows[i].SelfCycles,
			&rows[i].CumulativeCycles,
		)
	}

	settings := clickhousego.Settings{}
	if s.asyncInsertConfig.BusyTimeout > 0 {
		settings["async_insert_busy_timeout_ms"] = s.asyncInsertConfig.BusyTimeout.Milliseconds()
	}
	if s.asyncInsertConfig.MaxDataSize > 0 {
		settings["async_insert_max_data_size"] = s.asyncInsertConfig.MaxDataSize
	}
	if s.asyncInsertConfig.MaxQueryNumber > 0 {
		settings["async_insert_max_query_number"] = s.asyncInsertConfig.MaxQueryNumber
	}

	ctx = clickhousego.Context(
		ctx,
		clickhousego.WithAsync(true),
		clickhousego.WithSettings(settings),
	)

	if err := s.conn.Exec(
		ctx,
		fmt.Sprintf("INSERT INTO %s(generation, service, function, self_cycles, cumulative_cycles) VALUES %s", clusterTopTable, query.String()),
		args...,
	); err != nil {
		return fmt.Errorf("failed to execute async cluster top insert: %w", err)
	}
	return nil
}
