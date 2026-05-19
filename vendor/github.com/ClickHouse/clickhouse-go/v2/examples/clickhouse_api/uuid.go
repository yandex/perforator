package clickhouse_api

import (
	"context"
	"fmt"
	"github.com/google/uuid"
)

func UUIDInsertRead() error {
	conn, err := GetNativeConnection(nil, nil, nil)
	if err != nil {
		return err
	}
	ctx := context.Background()
	conn.Exec(ctx, "DROP TABLE IF EXISTS example")

	if err = conn.Exec(ctx, `
		CREATE TABLE example (
				col1 UUID,
				col2 UUID
			) 
			Engine Memory
		`); err != nil {
		return err
	}

	batch, err := conn.PrepareBatch(ctx, "INSERT INTO example")
	if err != nil {
		return err
	}
	col1Data, _ := uuid.NewUUID()
	if err = batch.Append(
		col1Data,
		"603966d6-ed93-11ec-8ea0-0242ac120002",
	); err != nil {
		return err
	}

	if err = batch.Send(); err != nil {
		return err
	}

	var (
		col1 uuid.UUID
		col2 uuid.UUID
	)

	if err = conn.QueryRow(ctx, "SELECT * FROM example").Scan(&col1, &col2); err != nil {
		return err
	}
	fmt.Printf("col1=%v, col2=%v\n", col1, col2)
	return nil
}
