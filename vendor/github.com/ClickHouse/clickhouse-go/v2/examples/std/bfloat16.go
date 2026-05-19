package std

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/ClickHouse/clickhouse-go/v2"
)

func BFloat16() error {
	conn, err := GetStdOpenDBConnection(clickhouse.Native, nil, nil, nil)
	if err != nil {
		return err
	}
	ctx := context.Background()
	_, err = conn.ExecContext(ctx, "DROP TABLE IF EXISTS example")
	if err != nil {
		return nil
	}

	const ddl = `
		CREATE TABLE example (
			  Col1 BFloat16,
			  Col2 Nullable(BFloat16)
		) Engine MergeTree() ORDER BY tuple()
		`

	if _, err := conn.ExecContext(ctx, ddl); err != nil {
		return nil
	}
	fmt.Println("Table created")

	scope, err := conn.Begin()
	if err != nil {
		return err
	}

	batch, err := scope.PrepareContext(ctx, "INSERT INTO example")
	if err != nil {
		return err
	}

	_, err = batch.ExecContext(ctx, float32(33.125), sql.NullFloat64{
		Float64: 34.25,
		Valid:   true,
	})
	if err != nil {
		return err
	}

	if err := scope.Commit(); err != nil {
		return err
	}

	fmt.Println("Values inserted")

	var (
		col1 float32
		col2 sql.NullFloat64
	)

	if err := conn.QueryRowContext(ctx, "SELECT * FROM example").Scan(&col1, &col2); err != nil {
		return nil
	}

	fmt.Printf("Col1: %v, Col2: %v\n", col1, col2)

	return nil
}
