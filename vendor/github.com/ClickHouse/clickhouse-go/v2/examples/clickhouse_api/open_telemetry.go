package clickhouse_api

import (
	"context"
	"fmt"
	"github.com/ClickHouse/clickhouse-go/v2"
	"go.opentelemetry.io/otel/trace"
)

func OpenTelemetry() error {
	conn, err := GetNativeConnection(nil, nil, nil)
	if err != nil {
		return err
	}
	var count uint64
	rows := conn.QueryRow(clickhouse.Context(context.Background(), clickhouse.WithSpan(
		trace.NewSpanContext(trace.SpanContextConfig{
			SpanID:  trace.SpanID{1, 2, 3, 4, 5},
			TraceID: trace.TraceID{5, 4, 3, 2, 1},
		}),
	)), "SELECT COUNT() FROM (SELECT number FROM system.numbers LIMIT 5)")
	if err := rows.Scan(&count); err != nil {
		return err
	}
	fmt.Printf("count: %d\n", count)
	return nil
}
