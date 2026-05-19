package clickhouse_api

import (
	"context"

	"github.com/ClickHouse/clickhouse-go/v2"
	clickhouse_tests "github.com/ClickHouse/clickhouse-go/v2/tests"
)

func AsyncInsertNative() error {
	conn, err := GetNativeConnection(nil, nil, nil)
	if err != nil {
		return err
	}
	ctx := context.Background()
	if !clickhouse_tests.CheckMinServerServerVersion(conn, 21, 12, 0) {
		return nil
	}
	defer func() {
		conn.Exec(ctx, "DROP TABLE example")
	}()
	conn.Exec(ctx, `DROP TABLE IF EXISTS example`)
	const ddl = `
		CREATE TABLE example (
			  Col1 UInt64
			, Col2 String
			, Col3 Array(UInt8)
			, Col4 DateTime
		) ENGINE = Memory
	`

	if err := conn.Exec(ctx, ddl); err != nil {
		return err
	}

	ctx = clickhouse.Context(ctx, clickhouse.WithAsync(false))
	{
		for i := 0; i < 100; i++ {
			err := conn.Exec(ctx, `INSERT INTO example VALUES (
				?, ?, ?, now()
			)`, i, "Golang SQL database driver", []uint8{1, 2, 3, 4, 5, 6, 7, 8, 9})
			if err != nil {
				return err
			}
		}
	}
	return nil
}
