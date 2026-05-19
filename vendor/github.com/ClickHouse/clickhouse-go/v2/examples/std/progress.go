package std

import (
	"context"
	"fmt"

	"github.com/ClickHouse/clickhouse-go/v2"
)

func ProgressProfileLogs() error {
	conn, err := GetStdOpenDBConnection(clickhouse.Native, clickhouse.Settings{
		"send_logs_level": "trace",
	}, nil, nil)
	if err != nil {
		return err
	}
	totalRows := uint64(0)
	// use context to pass a call back for progress and profile info
	ctx := clickhouse.Context(context.Background(), clickhouse.WithProgress(func(p *clickhouse.Progress) {
		fmt.Println("progress: ", p)
		totalRows += p.Rows
	}), clickhouse.WithProfileInfo(func(p *clickhouse.ProfileInfo) {
		fmt.Println("profile info: ", p)
	}), clickhouse.WithLogs(func(log *clickhouse.Log) {
		fmt.Println("log info: ", log)
	}))

	rows, err := conn.QueryContext(ctx, "SELECT number from numbers(1000000) LIMIT 1000000")
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
	}

	fmt.Printf("Total Rows: %d\n", totalRows)
	return rows.Err()
}
