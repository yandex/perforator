package clickhouse_api

import (
	"context"
	"fmt"
	"github.com/ClickHouse/clickhouse-go/v2"
)

func DynamicExample() error {
	ctx := context.Background()

	conn, err := GetNativeConnection(clickhouse.Settings{
		"allow_experimental_dynamic_type":                                   true,
		"output_format_native_use_flattened_dynamic_and_json_serialization": true,
	}, nil, nil)
	if err != nil {
		return err
	}

	if !CheckMinServerVersion(conn, 25, 6, 0) {
		fmt.Print("unsupported clickhouse version for Dynamic type")
		return nil
	}

	err = conn.Exec(ctx, "DROP TABLE IF EXISTS go_dynamic_example")
	if err != nil {
		return err
	}

	err = conn.Exec(ctx, `
		CREATE TABLE go_dynamic_example (
		    c Dynamic
		) ENGINE = Memory
	`)
	if err != nil {
		return err
	}

	batch, err := conn.PrepareBatch(ctx, "INSERT INTO go_dynamic_example (c)")
	if err != nil {
		return err
	}

	if err = batch.Append(true); err != nil {
		return err
	}

	if err = batch.Append(int64(42)); err != nil {
		return err
	}

	if err = batch.Append("example"); err != nil {
		return err
	}

	if err = batch.Append(clickhouse.NewDynamic("example dynamic")); err != nil {
		return err
	}

	if err = batch.Append(clickhouse.NewDynamicWithType("example dynamic with specific type", "String")); err != nil {
		return err
	}

	if err = batch.Append(nil); err != nil {
		return err
	}

	if err = batch.Send(); err != nil {
		return err
	}

	// Switch on Go Type

	rows, err := conn.Query(ctx, "SELECT c FROM go_dynamic_example")
	if err != nil {
		return err
	}

	for i := 0; rows.Next(); i++ {
		var row clickhouse.Dynamic
		err := rows.Scan(&row)
		if err != nil {
			return fmt.Errorf("failed to scan row index %d: %w", i, err)
		}

		switch row.Any().(type) {
		case bool:
			fmt.Printf("row at index %d is Bool: %v\n", i, row.Any())
		case int64:
			fmt.Printf("row at index %d is Int64: %v\n", i, row.Any())
		case string:
			fmt.Printf("row at index %d is String: %v\n", i, row.Any())
		case nil:
			fmt.Printf("row at index %d is NULL\n", i)
		}
	}

	// Switch on ClickHouse Type

	rows, err = conn.Query(ctx, "SELECT c FROM go_dynamic_example")
	if err != nil {
		return err
	}

	for i := 0; rows.Next(); i++ {
		var row clickhouse.Dynamic
		err := rows.Scan(&row)
		if err != nil {
			return fmt.Errorf("failed to scan row index %d: %w", i, err)
		}

		switch row.Type() {
		case "Bool":
			fmt.Printf("row at index %d is bool: %v\n", i, row.Any())
		case "Int64":
			fmt.Printf("row at index %d is int64: %v\n", i, row.Any())
		case "String":
			fmt.Printf("row at index %d is string: %v\n", i, row.Any())
		case "":
			fmt.Printf("row at index %d is nil\n", i)
		}
	}

	return nil
}
