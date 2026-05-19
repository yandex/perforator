package clickhouse_api

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/ClickHouse/clickhouse-go/v2"
)

func BFloat16() error {
	conn, err := GetNativeConnection(clickhouse.Settings{}, nil, nil)
	if err != nil {
		return err
	}
	ctx := context.Background()
	conn.Exec(ctx, "DROP TABLE IF EXISTS example")

	const ddl = `
		CREATE TABLE example (
			  Col1 BFloat16,
			  Col2 Nullable(BFloat16)
		) Engine MergeTree() ORDER BY tuple()
		`

	if err := conn.Exec(ctx, ddl); err != nil {
		return nil
	}
	fmt.Println("Table created")

	batch, err := conn.PrepareBatch(ctx, "INSERT INTO example")
	if err != nil {
		return err
	}
	batch.Append(float32(33.125), sql.NullFloat64{
		Float64: 34.25,
		Valid:   true,
	})

	fmt.Println("Rows to be inserted", batch.Rows())
	if err := batch.Send(); err != nil {
		return err
	}

	fmt.Printf("Inserted %d rows\n", batch.Rows())
	var (
		col1 float32
		col2 sql.NullFloat64
	)

	if err := conn.QueryRow(ctx, "SELECT * FROM example").Scan(&col1, &col2); err != nil {
		return nil
	}

	fmt.Printf("Col1: %v, Col2: %v\n", col1, col2)

	return nil
}
