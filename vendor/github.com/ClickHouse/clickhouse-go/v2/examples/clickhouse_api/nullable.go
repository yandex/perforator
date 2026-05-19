package clickhouse_api

import (
	"context"
	"database/sql"
	"fmt"
)

func NullableInsertRead() error {
	conn, err := GetNativeConnection(nil, nil, nil)
	if err != nil {
		return err
	}
	ctx := context.Background()
	conn.Exec(ctx, "DROP TABLE IF EXISTS example")

	if err = conn.Exec(ctx, `
		CREATE TABLE example (
				col1 Nullable(String),
				col2 String,
				col3 Nullable(Int8),
				col4 Nullable(Int64)
			) 
			Engine Memory
		`); err != nil {
		return err
	}

	batch, err := conn.PrepareBatch(ctx, "INSERT INTO example")
	if err != nil {
		return err
	}
	if err = batch.Append(
		nil,
		nil,
		nil,
		sql.NullInt64{Int64: 0, Valid: false},
	); err != nil {
		return err
	}

	if err = batch.Send(); err != nil {
		return err
	}

	var (
		col1 *string
		col2 string
		col3 *int8
		col4 sql.NullInt64
	)

	if err = conn.QueryRow(ctx, "SELECT * FROM example").Scan(&col1, &col2, &col3, &col4); err != nil {
		return err
	}
	fmt.Printf("col1=%v, col2=%v, col3=%v, col4=%v\n", col1, col2, col3, col4)
	return nil
}
