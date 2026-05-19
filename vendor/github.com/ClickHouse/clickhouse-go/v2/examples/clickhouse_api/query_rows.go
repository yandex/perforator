package clickhouse_api

import (
	"context"
	"fmt"
	"time"
)

func QueryRows() error {
	conn, err := GetNativeConnection(nil, nil, nil)
	if err != nil {
		return err
	}
	ctx := context.Background()
	defer func() {
		conn.Exec(ctx, "DROP TABLE example")
	}()
	if err := conn.Exec(ctx, `DROP TABLE IF EXISTS example`); err != nil {
		return err
	}
	err = conn.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS example (
			Col1 UInt8,
			Col2 String,
			Col3 DateTime
		) engine=Memory
	`)
	if err != nil {
		return err
	}
	batch, err := conn.PrepareBatch(ctx, "INSERT INTO example (Col1, Col2, Col3)")
	if err != nil {
		return err
	}
	for i := 0; i < 10; i++ {
		if err := batch.Append(uint8(i), fmt.Sprintf("value_%d", i), time.Now()); err != nil {
			return err
		}
	}
	if err := batch.Send(); err != nil {
		return err
	}

	rows, err := conn.Query(ctx, "SELECT Col1, Col2, Col3 FROM example WHERE Col1 >= 2")
	if err != nil {
		return err
	}
	for rows.Next() {
		var (
			col1 uint8
			col2 string
			col3 time.Time
		)
		if err := rows.Scan(&col1, &col2, &col3); err != nil {
			return err
		}
		fmt.Printf("row: col1=%d, col2=%s, col3=%s\n", col1, col2, col3)
	}
	rows.Close()
	return rows.Err()
}
