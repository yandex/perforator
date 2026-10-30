package aggregated

import (
	"context"

	clickhousego "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"

	"github.com/yandex/perforator/perforator/pkg/clickhouse"
	pb "github.com/yandex/perforator/perforator/proto/perforator"
)

func (s *ClickhouseAggregationStorage) QueryClusterTop(ctx context.Context, req *pb.ClusterTopQueryRequest) (*QueryResult, error) {
	builder, err := prepareQuery(req)
	if err != nil {
		return nil, err
	}
	sql, args, err := builder.ToSql()
	if err != nil {
		return nil, err
	}
	ctx = clickhousego.Context(ctx, clickhousego.WithSettings(s.config.Query.settings()))
	rows, err := clickhouse.QueryWithRetries(s.l, ctx, s.conn, sql, func(rows driver.Rows) (*QueryRow, error) {
		var row QueryRow
		if err := rows.ScanStruct(&row); err != nil {
			return nil, err
		}
		return &row, nil
	}, args...)
	if err != nil {
		return nil, err
	}
	limit := queryLimit(req)
	hasMore := uint64(len(rows)) > limit
	if hasMore {
		rows = rows[:limit]
	}
	return &QueryResult{Rows: rows, HasMore: hasMore}, nil
}
