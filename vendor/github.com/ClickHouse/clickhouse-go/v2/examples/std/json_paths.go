package std

import (
	"context"
	"fmt"
	"github.com/ClickHouse/clickhouse-go/v2"
	"time"
)

func JSONPathsExample() error {
	ctx := context.Background()

	conn, err := GetStdOpenDBConnection(clickhouse.Native, nil, nil, nil)
	if err != nil {
		return err
	}

	if !CheckMinServerVersion(conn, 25, 6, 0) {
		fmt.Print("unsupported clickhouse version for JSON type")
		return nil
	}

	_, err = conn.ExecContext(ctx, "SET allow_experimental_json_type = 1")
	if err != nil {
		return err
	}
	_, err = conn.ExecContext(ctx, "SET output_format_native_use_flattened_dynamic_and_json_serialization = 1")
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

	insertProduct := clickhouse.NewJSON()
	insertProduct.SetValueAtPath("id", clickhouse.NewDynamicWithType(uint64(1234), "UInt64"))
	insertProduct.SetValueAtPath("name", "Book")
	insertProduct.SetValueAtPath("tags", []string{"library", "fiction"})
	insertProduct.SetValueAtPath("pricing.price", int64(750))
	insertProduct.SetValueAtPath("pricing.currency", "usd")
	insertProduct.SetValueAtPath("metadata.region", "us")
	insertProduct.SetValueAtPath("metadata.page_count", int64(852))
	insertProduct.SetValueAtPath("created_at", clickhouse.NewDynamicWithType(time.Now().UTC().Truncate(time.Millisecond), "DateTime64(3)"))

	if _, err = batch.ExecContext(ctx, insertProduct); err != nil {
		return err
	}

	if err = tx.Commit(); err != nil {
		return err
	}

	var selectedProduct clickhouse.JSON

	if err = conn.QueryRowContext(ctx, "SELECT product FROM go_json_example").Scan(&selectedProduct); err != nil {
		return err
	}

	fmt.Printf("inserted product: %+v\n", insertProduct)
	fmt.Printf("selected product: %+v\n", selectedProduct)
	return nil
}
