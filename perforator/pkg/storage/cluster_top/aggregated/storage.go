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

type ClickhouseAggregationStorage struct {
	config Config
	l      xlog.Logger
	conn   *clickhouse.Connection
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

func NewStorage(l xlog.Logger, conn *clickhouse.Connection, config Config) *ClickhouseAggregationStorage {
	config.FillDefault()
	l = l.WithName("clustertop_clickhouse")

	return &ClickhouseAggregationStorage{
		l:      l,
		conn:   conn,
		config: config,
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

var likePatternEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

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
		From(clusterTopServingTable).
		Where("generation = ?", generation).
		Where("event_type = ?", clusterTopEventType).
		Where("sort_type = ?", "self_cycles")
	if optionsObject.function != "" {
		// Service drill-down denominators still use live source aggregates.
		builder = squirrel.Select("sum(self_cycles) as total_self_cycles").
			From(clusterTopByFunctionTable).
			Where("generation = ?", generation).
			Where("event_type = ?", clusterTopEventType).
			Where("function = ?", optionsObject.function)
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
	var orderByCycles, sortType string
	switch sortOrder {
	case SelfTimeSortOrder:
		orderByCycles = "cpu_cycles DESC"
		sortType = "self_cycles"
	case CumulativeTimeSortOrder:
		orderByCycles = "sum_cumulative_cycles DESC"
		sortType = "cumulative_cycles"
	default:
		return nil, fmt.Errorf("unknown sort order")
	}

	var builder squirrel.SelectBuilder
	switch aggregationType {
	case GroupByFunction:
		// Each sort copy already has globally aggregated counters. Reading in
		// key order avoids a full-generation GROUP BY and top-N sort.
		builder = squirrel.Select("function AS name, self_cycles AS cpu_cycles, cumulative_cycles AS sum_cumulative_cycles").
			From(clusterTopServingTable).
			Where("generation = ?", generation).
			Where("event_type = ?", clusterTopEventType).
			Where("sort_type = ?", sortType).
			OrderBy("sort_cycles DESC", "function DESC")
	case GroupByService:
		// Service contributions still need summing across buckets and unmerged
		// source rows; background merges are not required for correct results.
		builder = squirrel.Select(fmt.Sprintf("%s AS name, sum(self_cycles) AS cpu_cycles, sum(cumulative_cycles) AS sum_cumulative_cycles", groupBy)).
			From(clusterTopTable).
			Where("generation = ?", generation).
			Where("event_type = ?", clusterTopEventType).
			GroupBy(groupBy).
			OrderBy(orderByCycles, "service ASC")
	default:
		return nil, fmt.Errorf("unknown aggregation type: %s", aggregationType)
	}
	builder = builder.Limit(limit).Offset(offset)

	if filter != nil && filter.FunctionFilter != "" && filter.FunctionFilterMatchMode != "" {
		comparisonOperator := getComparisonOperator(filter.FunctionFilterMatchMode)
		if comparisonOperator == "" {
			return nil, fmt.Errorf("unknown function match mode: %s", filter.FunctionFilterMatchMode)
		}
		searchValue := filter.FunctionFilter
		if filter.FunctionFilterMatchMode == SubstringMatch {
			searchValue = "%" + likePatternEscaper.Replace(searchValue) + "%"
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
	return fmt.Sprintf("cluster-top:v3:%d:%d:primary-v2:0", generation, jobID)
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
	if len(result.Functions) == 0 {
		return nil
	}

	const valuesPlaceholder = "(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)"
	var query strings.Builder
	query.Grow(len(result.Functions) * (len(valuesPlaceholder) + 2))
	args := make([]any, 0, len(result.Functions)*11)
	for i := range result.Functions {
		function := &result.Functions[i]
		functionName := function.Name
		if len(functionName) > kMaxFunctionNameLength {
			functionName = functionName[:kMaxFunctionNameLength]
		}
		if i > 0 {
			query.WriteString(", ")
		}
		query.WriteString(valuesPlaceholder)
		args = append(args,
			result.Generation,
			bucket,
			clusterTopEventType,
			result.ServiceName,
			functionName,
			function.BuildID,
			function.SourceFile,
			function.SourceLine,
			function.SourceColumn,
			&function.SelfCycles,
			&function.CumulativeCycles,
		)
	}

	settings := clickhousego.Settings{
		"async_insert_deduplicate":   1,
		"insert_deduplication_token": insertDeduplicationToken(result.Generation, result.JobID),
	}
	if s.config.AsyncInsert.BusyTimeoutMin > 0 {
		settings["async_insert_busy_timeout_min_ms"] = s.config.AsyncInsert.BusyTimeoutMin.Milliseconds()
	}
	if s.config.AsyncInsert.BusyTimeoutMax > 0 {
		settings["async_insert_busy_timeout_max_ms"] = s.config.AsyncInsert.BusyTimeoutMax.Milliseconds()
	}
	if s.config.AsyncInsert.MaxDataSize > 0 {
		settings["async_insert_max_data_size"] = s.config.AsyncInsert.MaxDataSize
	}

	ctx = clickhousego.Context(
		ctx,
		clickhousego.WithAsync(true),
		clickhousego.WithSettings(settings),
	)

	// Language, binary_path and commit_id still use the table's defaults.
	if err := clickhouse.ExecWithRetries(
		s.l,
		ctx,
		s.conn,
		"cluster_top_v3_insert",
		fmt.Sprintf("INSERT INTO %s(generation, partition_bucket, event_type, service, function, build_id, source_file, source_line, source_column, self_cycles, cumulative_cycles) VALUES %s", clusterTopTable, query.String()),
		args...,
	); err != nil {
		return fmt.Errorf("failed to execute async cluster top insert: %w", err)
	}
	return nil
}
