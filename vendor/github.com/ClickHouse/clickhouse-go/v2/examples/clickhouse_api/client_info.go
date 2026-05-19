package clickhouse_api

import (
	"context"
	"github.com/ClickHouse/clickhouse-go/v2"
)

func ClientInfo() error {
	conn, err := clickhouse.Open(&clickhouse.Options{
		ClientInfo: clickhouse.ClientInfo{
			Products: []struct {
				Name    string
				Version string
			}{
				{Name: "my-app", Version: "0.1"},
			},
		},
	})
	if err != nil {
		return err
	}

	return conn.Exec(context.TODO(), "SELECT 1")
}
