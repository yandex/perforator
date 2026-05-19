package std

import (
	"context"
	"fmt"

	"github.com/ClickHouse/clickhouse-go/v2"
)

func JSONStringExample() error {
	ctx := context.Background()

	conn, err := GetStdOpenDBConnection(clickhouse.Native, nil, nil, nil)
	if err != nil {
		return err
	}

	if !CheckMinServerVersion(conn, 24, 10, 0) {
		fmt.Print("unsupported clickhouse version for JSON type")
		return nil
	}

	_, err = conn.ExecContext(ctx, "SET allow_experimental_json_type = 1")
	if err != nil {
		return err
	}

	_, err = conn.ExecContext(ctx, "SET output_format_native_write_json_as_string = 1")
	if err != nil {
		return err
	}

	defer func() {
		conn.Exec("DROP TABLE go_json_example")
	}()

	_, err = conn.ExecContext(ctx, "DROP TABLE IF EXISTS go_json_example")
	if err != nil {
		return err
	}

	_, err = conn.ExecContext(ctx, `
		CREATE TABLE go_json_example (
		    product JSON
		) ENGINE = Memory
	`)
	if err != nil {
		return err
	}

	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	batch, err := tx.PrepareContext(ctx, "INSERT INTO go_json_example (product)")
	if err != nil {
		return err
	}

	insertProductString := "{\"created_at\":\"2024-12-19T11:20:04.146Z\",\"id\":\"1234\"," +
		"\"metadata\":{\"page_count\":\"852\",\"region\":\"us\"},\"name\":\"Book\",\"pricing\":{\"currency\":\"usd\"," +
		"\"price\":\"750\"},\"tags\":[\"library\",\"fiction\"]}"

	if _, err = batch.ExecContext(ctx, insertProductString); err != nil {
		return err
	}

	if err = tx.Commit(); err != nil {
		return err
	}

	var selectedProductString string

	if err = conn.QueryRowContext(ctx, "SELECT product FROM go_json_example").Scan(&selectedProductString); err != nil {
		return err
	}

	fmt.Printf("inserted product string: %s\n", insertProductString)
	fmt.Printf("selected product string: %s\n", selectedProductString)
	fmt.Printf("inserted product string matches selected product string: %t\n", insertProductString == selectedProductString)
	return nil
}
