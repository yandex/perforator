package aggregated

import (
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"math"
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
	PartitionBucket  uint16  `ch:"partition_bucket"`
	EventType        string  `ch:"event_type"`
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
	BusyTimeoutMin time.Duration `yaml:"busy_timeout_min"`
	BusyTimeoutMax time.Duration `yaml:"busy_timeout_max"`
	MaxDataSize    uint64        `yaml:"max_data_size"`
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
	clusterTopTable           = "cluster_top_v3"
	clusterTopByFunctionTable = "cluster_top_by_function_v3"
	clusterTopEventType       = "cpu.cycles"
)

func (s *ClickhouseAggregationStorage) CountTotalSelfCycles(ctx context.Context, generation uint32, options ...CountTotalSelfCyclesOption) (*big.Int, error) {
	optionsObject := &totalSelfCyclesOptions{}
	for _, option := range options {
		option(optionsObject)
	}

	builder := squirrel.Select("sum(self_cycles) as total_self_cycles").
		From(clusterTopByFunctionTable).
		Where("generation = ?", generation).
		Where("event_type = ?", clusterTopEventType)
	if optionsObject.function != "" {
		builder = builder.Where("function = ?", optionsObject.function)
	}

	return s.countTotal(ctx, builder)
}

func (s *ClickhouseAggregationStorage) CountTotalCumulativeCycles(ctx context.Context, generation uint32, totalFunctionName string) (*big.Int, error) {
	builder := squirrel.Select("sum(cumulative_cycles) as total_cumulative_cycles").
		From(clusterTopByFunctionTable).
		Where("generation = ?", generation).
		Where("event_type = ?", clusterTopEventType)

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
	// Sum across buckets and unmerged rows; background merges are not required
	// for correct results from SummingMergeTree tables.

	builder := squirrel.
		Select(fmt.Sprintf("%s AS name, sum(self_cycles) AS cpu_cycles, sum(cumulative_cycles) as sum_cumulative_cycles", groupBy)).
		From(fromTable).
		Where("generation = ?", generation).
		Where("event_type = ?", clusterTopEventType).
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

// The hash and service encoding are part of the v3 storage contract. Changing
// either would route retries to different partitions and break deduplication.
func partitionBucket(service string, bucketCount uint16) (uint16, error) {
	if bucketCount == 0 {
		return 0, fmt.Errorf("partition bucket count must be positive")
	}
	hash := fnv.New64a()
	_, _ = hash.Write([]byte(service))
	return uint16(hash.Sum64() % uint64(bucketCount)), nil
}

// One job produces one logical source INSERT. The output identity must change
// if the computation or batching contract changes; attempts reuse this token.
func insertDeduplicationToken(generation int, jobID int64) string {
	return fmt.Sprintf("cluster-top:v3:%d:%d:primary-v1:0", generation, jobID)
}

func buildClusterTopRows(result *JobResult, bucket uint16) []clusterTopRow {
	if result == nil || len(result.Functions) == 0 {
		return nil
	}

	rows := make([]clusterTopRow, 0, len(result.Functions))
	for _, function := range result.Functions {
		functionName := function.Name
		if len(functionName) > kMaxFunctionNameLength {
			functionName = functionName[:kMaxFunctionNameLength]
		}

		rows = append(rows, clusterTopRow{
			Generation:       result.Generation,
			PartitionBucket:  bucket,
			EventType:        clusterTopEventType,
			Service:          result.ServiceName,
			Function:         functionName,
			SelfCycles:       function.SelfCycles,
			CumulativeCycles: function.CumulativeCycles,
		})
	}
	return rows
}

func (s *ClickhouseAggregationStorage) SaveClusterTopEntry(ctx context.Context, result *JobResult) error {
	if result == nil {
		return nil
	}
	if result.JobID <= 0 {
		return fmt.Errorf("invalid cluster top job ID: %d", result.JobID)
	}
	if result.Generation < 0 || uint64(result.Generation) > math.MaxUint32 {
		return fmt.Errorf("invalid cluster top generation: %d", result.Generation)
	}
	bucket, err := partitionBucket(result.ServiceName, result.BucketCount)
	if err != nil {
		return err
	}
	rows := buildClusterTopRows(result, bucket)
	if len(rows) == 0 {
		return nil
	}

	const valuesPlaceholder = "(?, ?, ?, ?, ?, ?, ?)"
	var query strings.Builder
	query.Grow(len(rows) * (len(valuesPlaceholder) + 2))
	args := make([]any, 0, len(rows)*7)
	for i := range rows {
		if i > 0 {
			query.WriteString(", ")
		}
		query.WriteString(valuesPlaceholder)
		args = append(args,
			rows[i].Generation,
			rows[i].PartitionBucket,
			rows[i].EventType,
			rows[i].Service,
			rows[i].Function,
			&rows[i].SelfCycles,
			&rows[i].CumulativeCycles,
		)
	}

	settings := clickhousego.Settings{
		"async_insert_deduplicate":   1,
		"insert_deduplication_token": insertDeduplicationToken(result.Generation, result.JobID),
	}
	if s.asyncInsertConfig.BusyTimeoutMin > 0 {
		settings["async_insert_busy_timeout_min_ms"] = s.asyncInsertConfig.BusyTimeoutMin.Milliseconds()
	}
	if s.asyncInsertConfig.BusyTimeoutMax > 0 {
		settings["async_insert_busy_timeout_max_ms"] = s.asyncInsertConfig.BusyTimeoutMax.Milliseconds()
	}
	if s.asyncInsertConfig.MaxDataSize > 0 {
		settings["async_insert_max_data_size"] = s.asyncInsertConfig.MaxDataSize
	}

	ctx = clickhousego.Context(
		ctx,
		clickhousego.WithAsync(true),
		clickhousego.WithSettings(settings),
	)

	// Omitted build, language and source dimensions use the table's empty/zero
	// defaults until they are available from the processing pipeline.
	if err := clickhouse.ExecWithRetries(
		s.l,
		ctx,
		s.conn,
		"cluster_top_v3_insert",
		fmt.Sprintf("INSERT INTO %s(generation, partition_bucket, event_type, service, function, self_cycles, cumulative_cycles) VALUES %s", clusterTopTable, query.String()),
		args...,
	); err != nil {
		return fmt.Errorf("failed to execute async cluster top insert: %w", err)
	}
	return nil
}
